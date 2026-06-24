package whatsmeow_service

import (
	"testing"

	"github.com/EvolutionAPI/evolution-go/pkg/config"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types"
)

func TestIncomingMessageFilterDropsGroupFromRawNode(t *testing.T) {
	service := whatsmeowService{
		config: &config.Config{EventIgnoreGroup: true},
	}
	filter := service.buildIncomingMessageFilter(nil)
	if filter == nil {
		t.Fatal("expected filter to be configured")
	}

	decision := filter(nil, &waBinary.Node{
		Attrs: waBinary.Attrs{
			"from": types.NewJID("123456789-111", types.GroupServer),
		},
	})

	if decision.Process {
		t.Fatal("expected raw group message to be filtered")
	}
	if decision.Reason != "ignored_group_raw" {
		t.Fatalf("expected ignored_group_raw reason, got %q", decision.Reason)
	}
}

func TestIncomingMessageFilterDropsParsedGroup(t *testing.T) {
	service := whatsmeowService{
		config: &config.Config{EventIgnoreGroup: true},
	}
	filter := service.buildIncomingMessageFilter(nil)
	if filter == nil {
		t.Fatal("expected filter to be configured")
	}

	decision := filter(&types.MessageInfo{
		MessageSource: types.MessageSource{
			Chat:    types.NewJID("123456789-111", types.GroupServer),
			IsGroup: true,
		},
	}, nil)

	if decision.Process {
		t.Fatal("expected parsed group message to be filtered")
	}
	if decision.Reason != "ignored_group" {
		t.Fatalf("expected ignored_group reason, got %q", decision.Reason)
	}
}

func TestIncomingMessageFilterAllowsPrivateMessage(t *testing.T) {
	service := whatsmeowService{
		config: &config.Config{EventIgnoreGroup: true},
	}
	filter := service.buildIncomingMessageFilter(nil)
	if filter == nil {
		t.Fatal("expected filter to be configured")
	}

	decision := filter(nil, &waBinary.Node{
		Attrs: waBinary.Attrs{
			"from": types.NewJID("5511999999999", types.DefaultUserServer),
		},
	})

	if !decision.Process {
		t.Fatalf("expected private message to be processed, got reason %q", decision.Reason)
	}
}
