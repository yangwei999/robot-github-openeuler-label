package main

import (
	"fmt"
	gc "github.com/opensourceways/community-robot-lib/githubclient"
	"strings"

	"github.com/opensourceways/community-robot-lib/utils"
	"k8s.io/apimachinery/pkg/util/sets"
)

type iRepoLabelHelper interface {
	getLabelsOfRepo() ([]string, error)
	isCollaborator(string) (bool, error)
	createLabelsOfRepo(missing []string) error
}

type repoLabelHelper struct {
	cli iClient
	p   gc.PRInfo
}

func (h *repoLabelHelper) isCollaborator(commenter string) (bool, error) {
	return h.cli.IsCollaborator(h.p, commenter)
}

func (h *repoLabelHelper) getLabelsOfRepo() ([]string, error) {
	labels, err := h.cli.GetRepositoryLabels(h.p)
	if err != nil {
		return nil, err
	}

	r := make([]string, len(labels))
	for _, item := range labels {
		r = append(r, item)
	}
	return r, nil
}

func (h *repoLabelHelper) createLabelsOfRepo(labels []string) error {
	mErr := utils.MultiError{}

	for _, v := range labels {
		if err := h.cli.CreateRepoLabel(h.p.Org, h.p.Repo, v); err != nil {
			mErr.AddError(err)
		}
	}

	return mErr.Err()
}

type labelHelper interface {
	addLabels([]string) error
	removeLabels([]string) error
	getCurrentLabels() sets.String
	addComment(string) error

	iRepoLabelHelper
}

type LabelHelper struct {
	*repoLabelHelper

	number int
	labels sets.String
}

func (h *LabelHelper) addLabels(label []string) error {
	for _, l := range label {
		if err := h.cli.AddPRLabel(h.p, l); err != nil {
			return err
		}
	}
	return nil
}

func (h *LabelHelper) removeLabels(label []string) error {
	for _, l := range label {
		fmt.Println("ready to remove label ", l)
		if err := h.cli.RemovePRLabel(h.p, l); err != nil {
			return err
		}
	}
	return nil
}

func (h *LabelHelper) getCurrentLabels() sets.String {
	return h.labels
}

func (h *LabelHelper) addComment(comment string) error {
	return h.cli.CreatePRComment(h.p, comment)
}

type labelSet struct {
	m map[string]string
	s sets.String
}

func (ls *labelSet) count() int {
	return len(ls.m)
}

func (ls *labelSet) toList() []string {
	return ls.s.UnsortedList()
}

func (ls *labelSet) origin(data []string) []string {
	r := make([]string, 0, len(data))
	for _, item := range data {
		if v, ok := ls.m[item]; ok {
			r = append(r, v)
		}
	}
	return r
}

func (ls *labelSet) intersection(ls1 *labelSet) []string {
	return ls.s.Intersection(ls1.s).UnsortedList()
}

func (ls *labelSet) difference(ls1 *labelSet) []string {
	return ls.s.Difference(ls1.s).UnsortedList()
}

func newLabelSet(data []string) *labelSet {
	m := map[string]string{}
	v := make([]string, len(data))
	for i := range data {
		v[i] = strings.ToLower(data[i])
		m[v[i]] = data[i]
	}

	return &labelSet{
		m: m,
		s: sets.NewString(v...),
	}
}
