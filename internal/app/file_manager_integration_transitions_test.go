package app

import (
	"context"
	"testing"
)

type integrationProbe struct {
	state       FileManagerIntegrationState
	calls       int
	staysBroken bool
}

func (p *integrationProbe) inspect(context.Context) (FileManagerIntegrationState, error) {
	return p.state, nil
}
func (p *integrationProbe) act(context.Context, string) error {
	p.calls++
	if !p.staysBroken {
		p.state.Status = IntegrationRegistered
	}
	return nil
}
func TestFileManagerIntegrationRepairIsExplicitAndVerified(t *testing.T) {
	for _, broken := range []bool{false, true} {
		t.Run(map[bool]string{false: "restored", true: "still-broken"}[broken], func(t *testing.T) {
			p := &integrationProbe{state: FileManagerIntegrationState{Status: IntegrationNeedsRepair, Actions: []string{"repair"}}, staysBroken: broken}
			s := FileManagerIntegrationService{platform: "darwin", adapter: p}
			if _, err := s.Inspect(t.Context()); err != nil {
				t.Fatal(err)
			}
			if p.calls != 0 {
				t.Fatal("passive inspection mutated registration")
			}
			state, err := s.Act(t.Context(), "repair")
			if broken {
				if err == nil || state.Status == IntegrationRegistered {
					t.Fatal("failed reinspection reported Repair success")
				}
			} else if err != nil || state.Status != IntegrationRegistered {
				t.Fatalf("Repair = %#v, %v", state, err)
			}
			if p.calls != 1 {
				t.Fatalf("Repair calls = %d, want one", p.calls)
			}
		})
	}
}
