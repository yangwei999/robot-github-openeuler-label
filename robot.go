package main

import (
	"fmt"
	sdk "github.com/google/go-github/v36/github"
	"github.com/opensourceways/community-robot-lib/config"
	gc "github.com/opensourceways/community-robot-lib/githubclient"
	framework "github.com/opensourceways/community-robot-lib/robot-github-framework"

	"github.com/opensourceways/community-robot-lib/utils"
	"github.com/sirupsen/logrus"
)

const (
	botName             = "label"
	actionOpen          = "open"
	sourceBranchChanged = "synchronize"
)

type iClient interface {
	AddPRLabel(pr gc.PRInfo, label string) error
	RemovePRLabel(pr gc.PRInfo, label string) error
	GetPRCommits(pr gc.PRInfo) ([]*sdk.RepositoryCommit, error)
	CreatePRComment(pr gc.PRInfo, comment string) error
	GetPRLabels(pr gc.PRInfo) ([]string, error)
	IsCollaborator(pr gc.PRInfo, login string) (bool, error)
	GetRepositoryLabels(pr gc.PRInfo) ([]string, error)
	CreateRepoLabel(org, repo, label string) error
}

func newRobot(cli iClient) *robot {
	return &robot{cli: cli}
}

type robot struct {
	cli iClient
}

func (bot *robot) NewConfig() config.Config {
	return &configuration{}
}

func (bot *robot) getConfig(cfg config.Config, org, repo string) (*botConfig, error) {
	c, ok := cfg.(*configuration)
	if !ok {
		return nil, fmt.Errorf("can't convert to configuration")
	}

	if bc := c.configFor(org, repo); bc != nil {
		return bc, nil
	}

	return nil, fmt.Errorf("no config for this repo:%s/%s", org, repo)
}

func (bot *robot) RegisterEventHandler(p framework.HandlerRegister) {
	p.RegisterPullRequestHandler(bot.handlePREvent)
	p.RegisterIssueCommentHandler(bot.HandleCommentEvent)
}

func (bot *robot) handlePREvent(e *sdk.PullRequestEvent, pc config.Config, log *logrus.Entry) error {
	org, repo := gc.GetOrgRepo(e.GetRepo())

	c, err := bot.getConfig(pc, org, repo)
	pull := gc.PRInfo{
		Org:    org,
		Repo:   repo,
		Number: *e.Number,
	}

	merr := utils.NewMultiErrors()
	if err = bot.handleClearLabel(e, c, pull); err != nil {
		merr.AddError(err)
	}

	commits, err := bot.cli.GetPRCommits(pull)
	if err != nil {
		merr.AddError(err)
	}

	commitsCount := len(commits)

	if err = bot.handleSquashLabel(e, pull, commitsCount, c.SquashConfig); err != nil {
		merr.AddError(err)
	}

	return merr.Err()
}

func (bot *robot) HandleCommentEvent(e *sdk.IssueCommentEvent, pc config.Config, log *logrus.Entry) error {
	if e.GetAction() != "created" || e.Issue.GetState() != actionOpen {
		log.Debug("Event is not a creation of a comment or PR is not opened, skipping.")
		return nil
	}

	org, repo := gc.GetOrgRepo(e.GetRepo())
	c, err := bot.getConfig(pc, org, repo)
	if err != nil {
		return err
	}

	info := gc.PRInfo{Org: org, Repo: repo, Number: e.GetIssue().GetNumber()}

	toAdd, toRemove := getMatchedLabels(e.GetComment().GetBody())
	if len(toAdd) == 0 && len(toRemove) == 0 {
		log.Debug("invalid comment, skipping.")
		return nil
	}

	return bot.handleLabels(e, info, toAdd, toRemove, c, log)
}
