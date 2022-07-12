package main

import (
	sdk "github.com/google/go-github/v36/github"
	gc "github.com/opensourceways/community-robot-lib/githubclient"
	"k8s.io/apimachinery/pkg/util/sets"
)

func (bot *robot) handleSquashLabel(e *sdk.PullRequestEvent, p gc.PRInfo, commits int, cfg SquashConfig) error {
	if cfg.unableCheckingSquash() {
		return nil
	}

	if e.GetPullRequest().GetState() != actionOpen && e.GetAction() != sourceBranchChanged {
		return nil
	}

	labelSet := sets.NewString()
	for _, v := range e.GetPullRequest().Labels {
		labelSet.Insert(*v.Name)
	}
	hasSquashLabel := labelSet.Has(cfg.SquashCommitLabel)
	exceeded := commits > cfg.CommitsThreshold

	if exceeded && !hasSquashLabel {
		return bot.cli.AddPRLabel(p, cfg.SquashCommitLabel)
	}

	if !exceeded && hasSquashLabel {
		return bot.cli.RemovePRLabel(p, cfg.SquashCommitLabel)
	}

	return nil
}
