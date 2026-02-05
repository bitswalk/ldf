package cmd

import (
	"context"

	"github.com/spf13/cobra"
)

// completionDistributionIDs returns a ValidArgsFunction that completes distribution IDs
func completionDistributionIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	c := getClient()
	ctx := context.Background()
	resp, err := c.ListDistributions(ctx, nil)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	suggestions := make([]string, len(resp.Distributions))
	for i, d := range resp.Distributions {
		suggestions[i] = d.ID + "\t" + d.Name
	}
	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

// completionComponentIDs returns a ValidArgsFunction that completes component IDs
func completionComponentIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	c := getClient()
	ctx := context.Background()
	resp, err := c.ListComponents(ctx, nil)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	suggestions := make([]string, len(resp.Components))
	for i, comp := range resp.Components {
		suggestions[i] = comp.ID + "\t" + comp.Name
	}
	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

// completionSourceIDs returns a ValidArgsFunction that completes source IDs
func completionSourceIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	c := getClient()
	ctx := context.Background()
	resp, err := c.ListSources(ctx, nil)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	suggestions := make([]string, len(resp.Sources))
	for i, s := range resp.Sources {
		suggestions[i] = s.ID + "\t" + s.Name
	}
	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

// completionRoleIDs returns a ValidArgsFunction that completes role IDs
func completionRoleIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	c := getClient()
	ctx := context.Background()
	resp, err := c.ListRoles(ctx)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	suggestions := make([]string, len(resp.Roles))
	for i, r := range resp.Roles {
		suggestions[i] = r.ID + "\t" + r.Name
	}
	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

// completionOutputFormat provides completion for --output flag
func completionOutputFormat(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"table", "json", "yaml"}, cobra.ShellCompDirectiveNoFileComp
}

// completionDistributionStatus provides completion for --status flag
func completionDistributionStatus(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"pending", "downloading", "validating", "ready", "failed"}, cobra.ShellCompDirectiveNoFileComp
}

// completionVisibility provides completion for --visibility flag
func completionVisibility(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"public", "private"}, cobra.ShellCompDirectiveNoFileComp
}

// completionCategories provides completion for --category flag
func completionCategories(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	c := getClient()
	ctx := context.Background()
	resp, err := c.ListCategories(ctx)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	suggestions := make([]string, 0, len(resp))
	for cat := range resp {
		suggestions = append(suggestions, cat)
	}
	return suggestions, cobra.ShellCompDirectiveNoFileComp
}
