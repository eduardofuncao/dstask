package github

import (
	"bytes"
	"crypto/md5"
	"hash"
	"io"
	"strconv"
	"time"

	"github.com/gofrs/uuid"
	"github.com/naggie/dstask"
)

// IssueData is a compact way to represent an issue
// so that templates can be expanded simply (without nested properties).
type IssueData struct {
	// internal properties
	uuidHash hash.Hash     // to generate UUID's
	buf      *bytes.Buffer // to expand templates into

	// populated from our hash
	uuid uuid.UUID
	UUID string

	// populated from our scraping config
	RepoOwner string
	RepoName  string

	// populated from the data GitHub returned to us
	Author    string
	Body      string
	ClosedAt  time.Time
	Closed    bool
	CreatedAt time.Time
	Milestone string
	Number    int
	State     string
	Title     string
	URL       string
}

func NewIssueData() *IssueData {
	return &IssueData{
		// to write key issue features into, to generate the UUID
		uuidHash: md5.New(),
		buf:      &bytes.Buffer{},
	}
}

// Init sets all properties to match the given repo owner, name and Github data.
func (id *IssueData) Init(repoOwner, repoName string, i Issue) {
	id.RepoOwner = repoOwner
	id.RepoName = repoName
	id.Author = i.Author.Name
	id.Body = i.Body
	id.ClosedAt = i.ClosedAt
	id.Closed = i.Closed
	id.CreatedAt = i.CreatedAt
	id.Milestone = i.Milestone.Title
	id.Number = i.Number
	id.State = i.State
	id.Title = i.Title
	id.URL = i.URL

	id.uuidHash.Reset()
	_, _ = io.WriteString(id.uuidHash, "GH")
	_, _ = io.WriteString(id.uuidHash, "\x00")
	_, _ = io.WriteString(id.uuidHash, repoOwner)
	_, _ = io.WriteString(id.uuidHash, "\x00")
	_, _ = io.WriteString(id.uuidHash, repoName)
	_, _ = io.WriteString(id.uuidHash, "\x00")
	_, _ = io.WriteString(id.uuidHash, strconv.Itoa(i.Number))
	id.uuidHash.Sum(id.uuid[:0])
	id.UUID = id.uuid.String()
}

// ToTask generates a Task based on the issue data.
func (id *IssueData) ToTask(templates Templates) (dstask.Task, error) {
	task := dstask.Task{
		UUID:    id.UUID,
		Status:  dstask.STATUS_PENDING,
		Created: id.CreatedAt,
	}

	if id.Closed {
		task.Status = dstask.STATUS_RESOLVED
		task.Resolved = id.ClosedAt
	}

	err := templates.Summary.Execute(id.buf, id)
	if err != nil {
		return task, err
	}

	task.Summary = id.buf.String()
	id.buf.Reset()

	err = templates.Project.Execute(id.buf, id)
	if err != nil {
		return task, err
	}

	task.Project = id.buf.String()
	id.buf.Reset()

	err = templates.Priority.Execute(id.buf, id)
	if err != nil {
		return task, err
	}

	task.Priority = id.buf.String()
	id.buf.Reset()

	err = templates.Notes.Execute(id.buf, id)
	if err != nil {
		return task, err
	}

	task.Notes = id.buf.String()
	id.buf.Reset()

	for _, t := range templates.Tags {
		err = t.Execute(id.buf, id)
		if err != nil {
			return task, err
		}

		if id.buf.String() != "" {
			task.Tags = append(task.Tags, id.buf.String())
		}

		id.buf.Reset()
	}

	return task, nil
}
