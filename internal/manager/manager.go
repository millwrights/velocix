package manager

import (
	"context"
	"log/slog"
	"sync"

	"github.com/skalluru/velocix/internal/config"
	gh "github.com/skalluru/velocix/internal/github"
	"github.com/skalluru/velocix/internal/poller"
	"github.com/skalluru/velocix/internal/store"
)

type OrgInstance struct {
	Config config.OrgConfig
	Store  *store.Store
	Poller *poller.Poller
	Cancel context.CancelFunc
}

type Manager struct {
	mu        sync.RWMutex
	cfg       *config.Config
	instances map[string]*OrgInstance
	activeOrg string
	logger    *slog.Logger
	parentCtx context.Context
}

func New(ctx context.Context, cfg *config.Config, logger *slog.Logger) *Manager {
	m := &Manager{
		cfg:       cfg,
		instances: make(map[string]*OrgInstance),
		activeOrg: cfg.ActiveOrg,
		logger:    logger,
		parentCtx: ctx,
	}

	// Start pollers for all configured orgs
	for _, orgCfg := range cfg.Orgs {
		m.startOrg(orgCfg)
	}

	return m
}

func (m *Manager) startOrg(orgCfg config.OrgConfig) {
	ctx, cancel := context.WithCancel(m.parentCtx)
	client := gh.NewClient(orgCfg.GitHubToken, m.logger)
	st := store.NewStore(m.cfg.DataDir, orgCfg.Organization, m.logger)
	p := poller.New(client, st, orgCfg.Organization, m.cfg.PollInterval, m.logger)
	p.Start(ctx)

	m.instances[orgCfg.Name] = &OrgInstance{
		Config: orgCfg,
		Store:  st,
		Poller: p,
		Cancel: cancel,
	}
}

func (m *Manager) GetActiveStore() *store.Store {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if inst, ok := m.instances[m.activeOrg]; ok {
		return inst.Store
	}
	// Fallback to first
	for _, inst := range m.instances {
		return inst.Store
	}
	return nil
}

func (m *Manager) GetActiveOrg() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeOrg
}

func (m *Manager) SetActiveOrg(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.instances[name]; ok {
		m.activeOrg = name
		m.cfg.ActiveOrg = name
		m.cfg.Save()
		return true
	}
	// Try matching by organization name
	for key, inst := range m.instances {
		if inst.Config.Organization == name {
			m.activeOrg = key
			m.cfg.ActiveOrg = key
			m.cfg.Save()
			return true
		}
	}
	return false
}

func (m *Manager) ListOrgs() []OrgInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var orgs []OrgInfo
	for _, inst := range m.instances {
		orgs = append(orgs, OrgInfo{
			Name:         inst.Config.Name,
			Organization: inst.Config.Organization,
			Active:       inst.Config.Name == m.activeOrg,
			RunCount:     len(inst.Store.GetAll()),
		})
	}
	return orgs
}

type OrgInfo struct {
	Name         string `json:"name"`
	Organization string `json:"organization"`
	Active       bool   `json:"active"`
	RunCount     int    `json:"run_count"`
}

func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, inst := range m.instances {
		inst.Cancel()
	}
}
