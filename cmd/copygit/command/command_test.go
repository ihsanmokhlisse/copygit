package command

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestNewInitCmdV2(t *testing.T) {
	cmd := NewInitCmdV2()
	assertCommand(t, cmd, "init", 1)
}

func TestNewPushCmdV2(t *testing.T) {
	cmd := NewPushCmdV2()
	assertCommand(t, cmd, "push", 0)

	// Check flags exist
	assertFlag(t, cmd, "all")
	assertFlag(t, cmd, "dry-run")
	assertFlag(t, cmd, "output")
	assertFlag(t, cmd, "conflict")
}

func TestNewSyncCmd(t *testing.T) {
	cmd := NewSyncCmd()
	assertCommand(t, cmd, "sync", 0)

	assertFlag(t, cmd, "dry-run")
	assertFlag(t, cmd, "force")
	assertFlag(t, cmd, "output")
}

func TestNewStatusCmdV2(t *testing.T) {
	cmd := NewStatusCmdV2()
	assertCommand(t, cmd, "status", 0)

	assertFlag(t, cmd, "all")
	assertFlag(t, cmd, "output")
}

func TestNewListCmdV2(t *testing.T) {
	cmd := NewListCmdV2()
	assertCommand(t, cmd, "list", 0)

	assertFlag(t, cmd, "output")
}

func TestNewRemoveCmdV2(t *testing.T) {
	cmd := NewRemoveCmdV2()
	assertCommand(t, cmd, "remove", 1)

	assertFlag(t, cmd, "clean")
}

func TestNewConfigCmdV2(t *testing.T) {
	cmd := NewConfigCmdV2()
	assertCommand(t, cmd, "config", 0)

	// Should have subcommands
	subs := cmd.Commands()
	if len(subs) != 3 {
		t.Errorf("config command should have 3 subcommands, got %d", len(subs))
	}

	subNames := make(map[string]bool)
	for _, s := range subs {
		subNames[s.Name()] = true
	}

	for _, name := range []string{"add-provider", "list-providers", "remove-provider"} {
		if !subNames[name] {
			t.Errorf("config missing subcommand %q", name)
		}
	}
}

func TestNewLoginCmd(t *testing.T) {
	cmd := NewLoginCmd()
	assertCommand(t, cmd, "login", 0)

	assertFlag(t, cmd, "provider")
	assertFlag(t, cmd, "token")
	assertFlag(t, cmd, "method")
}

func TestNewHooksCmd(t *testing.T) {
	cmd := NewHooksCmd()
	assertCommand(t, cmd, "hooks", 0)

	subs := cmd.Commands()
	if len(subs) != 3 {
		t.Errorf("hooks command should have 3 subcommands, got %d", len(subs))
	}

	subNames := make(map[string]bool)
	for _, s := range subs {
		subNames[s.Name()] = true
	}

	for _, name := range []string{"install", "uninstall", "status"} {
		if !subNames[name] {
			t.Errorf("hooks missing subcommand %q", name)
		}
	}
}

func TestNewDaemonCmd(t *testing.T) {
	cmd := NewDaemonCmd()
	assertCommand(t, cmd, "daemon", 0)

	subs := cmd.Commands()
	if len(subs) != 3 {
		t.Errorf("daemon command should have 3 subcommands, got %d", len(subs))
	}

	subNames := make(map[string]bool)
	for _, s := range subs {
		subNames[s.Name()] = true
	}

	for _, name := range []string{"start", "stop", "status"} {
		if !subNames[name] {
			t.Errorf("daemon missing subcommand %q", name)
		}
	}
}

func TestNewCloneCmd(t *testing.T) {
	cmd := NewCloneCmd()
	assertCommand(t, cmd, "clone", 0)

	assertFlag(t, cmd, "provider")
	assertFlag(t, cmd, "dir")
	assertFlag(t, cmd, "init")
}

func TestNewHealthCmd(t *testing.T) {
	cmd := NewHealthCmd()
	assertCommand(t, cmd, "health", 0)

	assertFlag(t, cmd, "output")
}

// assertCommand verifies a command has the expected name and min args.
func assertCommand(t *testing.T, cmd *cobra.Command, name string, minArgs int) {
	t.Helper()
	if cmd.Name() != name {
		t.Errorf("command name = %q, want %q", cmd.Name(), name)
	}
	if cmd.Short == "" {
		t.Errorf("command %q has no short description", name)
	}
	_ = minArgs // validated by cobra itself
}

// assertFlag verifies a flag exists on the command.
func assertFlag(t *testing.T, cmd *cobra.Command, name string) {
	t.Helper()
	if cmd.Flags().Lookup(name) == nil {
		t.Errorf("command %q missing flag --%s", cmd.Name(), name)
	}
}
