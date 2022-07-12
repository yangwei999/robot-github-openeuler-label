package main

import (
	"fmt"
	sdk "github.com/google/go-github/v36/github"
	gc "github.com/opensourceways/community-robot-lib/githubclient"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
)

func (bot *robot) handleClearLabel(e *sdk.PullRequestEvent, cfg *botConfig, p gc.PRInfo) error {
	if e.GetAction() != sourceBranchChanged {
		return nil
	}

	labels := sets.NewString()
	for _, v := range e.GetPullRequest().Labels {
		labels.Insert(*v.Name)
	}
	toRemove := getClearLabels(labels, cfg)
	if len(toRemove) == 0 {
		return nil
	}

	for _, l := range toRemove {
		if err := bot.cli.RemovePRLabel(p, l); err != nil {
			return err
		}
	}

	comment := fmt.Sprintf(
		"This pull request source branch has changed, so removes the following label(s): %s.",
		strings.Join(toRemove, ", "),
	)

	return bot.cli.CreatePRComment(p, comment)
}

func getClearLabels(labels sets.String, cfg *botConfig) []string {
	var r []string

	all := labels
	if len(cfg.ClearLabels) > 0 {
		v := all.Intersection(sets.NewString(cfg.ClearLabels...))
		if v.Len() > 0 {
			r = v.UnsortedList()
			all = all.Difference(v)
		}
	}

	exp := cfg.clearLabelsByRegexp
	if exp != nil {
		for k := range all {
			if exp.MatchString(k) {
				r = append(r, k)
			}
		}
	}

	return r
}
