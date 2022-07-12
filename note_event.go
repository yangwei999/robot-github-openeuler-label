package main

import (
	"fmt"
	sdk "github.com/google/go-github/v36/github"
	gc "github.com/opensourceways/community-robot-lib/githubclient"
	"github.com/opensourceways/community-robot-lib/utils"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"
	"strings"
)

func (bot *robot) handleLabels(
	e *sdk.IssueCommentEvent,
	i gc.PRInfo,
	toAdd []string,
	toRemove []string,
	cfg *botConfig,
	log *logrus.Entry,
) error {
	lh := genLabelHelper(e, i, bot.cli)
	if lh == nil {
		return nil
	}

	add := newLabelSet(toAdd)
	remove := newLabelSet(toRemove)
	fmt.Println("remove labels ", remove)
	if v := add.intersection(remove); len(v) > 0 {
		return lh.addComment(fmt.Sprintf(
			"conflict labels(%s) exit", strings.Join(add.origin(v), ", "),
		))
	}

	merr := utils.NewMultiErrors()

	if remove.count() > 0 {
		if _, err := removeLabels(lh, remove); err != nil {
			merr.AddError(err)
		}
	}

	author := e.GetComment().GetUser().GetLogin()

	if add.count() > 0 {
		err := addLabels(lh, add, author, cfg, log)
		if err != nil {
			merr.AddError(err)
		}
	}
	return merr.Err()
}

func genLabelHelper(e *sdk.IssueCommentEvent, pi gc.PRInfo, cli iClient) labelHelper {
	rlh := &repoLabelHelper{
		cli: cli,
		p:   pi,
	}

	lbs := make([]string, len(e.GetIssue().Labels))
	for _, l := range e.GetIssue().Labels {
		lbs = append(lbs, *l.Name)
	}

	if len(lbs) == 0 {
		return &LabelHelper{
			number:          e.GetIssue().GetNumber(),
			labels:          sets.NewString(),
			repoLabelHelper: rlh,
		}
	}

	return &LabelHelper{
		number:          e.GetIssue().GetNumber(),
		labels:          sets.NewString(lbs...),
		repoLabelHelper: rlh,
	}
}

func addLabels(lh labelHelper, toAdd *labelSet, commenter string, cfg *botConfig, log *logrus.Entry) error {
	canAdd, missing, err := checkLabelsToAdd(lh, toAdd, commenter, cfg, log)
	if err != nil {
		return err
	}

	merr := utils.NewMultiErrors()

	if len(canAdd) > 0 {
		ls := sets.NewString(canAdd...).Difference(lh.getCurrentLabels())
		if ls.Len() > 0 {
			if err := lh.addLabels(ls.UnsortedList()); err != nil {
				merr.AddError(err)
			}
		}
	}

	if len(missing) > 0 {
		msg := fmt.Sprintf(
			"The label(s) `%s` cannot be applied, because the repository doesn't have them",
			strings.Join(missing, ", "),
		)

		if err := lh.addComment(msg); err != nil {
			merr.AddError(err)
		}
	}

	return merr.Err()
}

func checkLabelsToAdd(
	h labelHelper,
	toAdd *labelSet,
	commenter string,
	cfg *botConfig,
	log *logrus.Entry,
) ([]string, []string, error) {
	v, err := h.getLabelsOfRepo()
	if err != nil {
		return nil, nil, err
	}
	repoLabels := newLabelSet(v)

	missing := toAdd.difference(repoLabels)
	if len(missing) == 0 {
		return repoLabels.origin(toAdd.toList()), nil, nil
	}

	var canAdd []string
	if len(missing) < toAdd.count() {
		canAdd = repoLabels.origin(toAdd.intersection(repoLabels))
	}

	missing = toAdd.origin(missing)

	if !cfg.AllowCreatingLabelsByCollaborator {
		return canAdd, missing, nil
	}

	b, err := h.isCollaborator(commenter)
	if err != nil {
		return nil, nil, err
	}
	if b {
		if err := h.createLabelsOfRepo(missing); err != nil {
			log.Error(err)
		}

		return append(canAdd, missing...), nil, nil
	}
	return canAdd, missing, nil
}

func removeLabels(lh labelHelper, toRemove *labelSet) ([]string, error) {
	v, err := lh.getLabelsOfRepo()
	if err != nil {
		return nil, err
	}
	repoLabels := newLabelSet(v)

	ls := lh.getCurrentLabels().Intersection(sets.NewString(
		repoLabels.origin(toRemove.toList())...)).UnsortedList()

	if len(ls) == 0 {
		return nil, nil
	}
	return ls, lh.removeLabels(ls)
}
