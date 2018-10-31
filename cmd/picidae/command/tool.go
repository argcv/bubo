package command

import (
	"github.com/spf13/cobra"
)

func NewSSLCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "ssl",
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			return err
		},
	}

	return cmd
}

func NewToolCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "tool",
		PreRunE: func(cmd *cobra.Command, args []string) (err error) {
			return err
		},
	}
	cmd.AddCommand(
		NewSSLCommand(),
	)
	return cmd
}
