package gcs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"testing"

	gcsapi "cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"

	"github.com/nkoji21/leetdaily/internal/config"
	"github.com/nkoji21/leetdaily/internal/problemcache"
	"github.com/nkoji21/leetdaily/internal/storage"
	"github.com/nkoji21/leetdaily/internal/storage/repositorytest"
)

func TestRepositorySuite(t *testing.T) {
	t.Parallel()

	repositorytest.RunRepositorySuite(t, "gcs", func(t *testing.T) repositorytest.Harness {
		client := newFakeClient()
		repository, err := New(client, "leetdaily-test", storage.Paths{
			ConfigPath:   "config.json",
			GuildsPath:   "guilds.json",
			StatePath:    "state.json",
			ProblemsPath: "problems.json",
		})
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		return repositorytest.Harness{
			Repository: repository,
			SeedConfig: func(tb testing.TB, cfg config.Config) {
				tb.Helper()

				data, err := json.MarshalIndent(cfg, "", "  ")
				if err != nil {
					tb.Fatalf("json.MarshalIndent() error = %v", err)
				}

				if _, err := client.WriteObject(context.Background(), "leetdaily-test", "config.json", append(data, '\n'), WriteObjectOptions{DoesNotExist: true}); err != nil {
					tb.Fatalf("WriteObject(config) error = %v", err)
				}
			},
			SeedGuildSettings: func(tb testing.TB, guilds config.GuildSettings) {
				tb.Helper()

				data, err := json.MarshalIndent(guilds, "", "  ")
				if err != nil {
					tb.Fatalf("json.MarshalIndent() error = %v", err)
				}

				if _, err := client.WriteObject(context.Background(), "leetdaily-test", "guilds.json", append(data, '\n'), WriteObjectOptions{DoesNotExist: true}); err != nil {
					tb.Fatalf("WriteObject(guilds) error = %v", err)
				}
			},
		}
	})
}

func TestNormalizeErrorMapsGenerationConflicts(t *testing.T) {
	t.Parallel()

	err := normalizeError(&googleapi.Error{Code: 412}, "state.json")
	if !errors.Is(err, storage.ErrConflict) {
		t.Fatalf("normalizeError(412) = %v, want ErrConflict", err)
	}
}

func TestWriteUsesGenerationPreconditions(t *testing.T) {
	t.Parallel()

	client := newFakeClient()
	repository, err := New(client, "leetdaily-test", storage.Paths{
		ConfigPath:   "config.json",
		GuildsPath:   "guilds.json",
		StatePath:    "state.json",
		ProblemsPath: "problems.json",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	version, err := repository.SaveProblemCache(context.Background(), emptyProblemCache(), storage.Version{})
	if err != nil {
		t.Fatalf("SaveProblemCache(create) error = %v", err)
	}

	if got := client.lastWriteOptions("problems.json"); !got.DoesNotExist {
		t.Fatalf("initial write options = %#v, want DoesNotExist=true", got)
	}

	_, err = repository.SaveProblemCache(context.Background(), emptyProblemCache(), version)
	if err != nil {
		t.Fatalf("SaveProblemCache(update) error = %v", err)
	}

	if got := client.lastWriteOptions("problems.json"); got.MatchGeneration < 1 {
		t.Fatalf("update write options = %#v, want MatchGeneration > 0", got)
	}
}

func emptyProblemCache() problemcache.Cache {
	return problemcache.Cache{Problems: []problemcache.Problem{}}
}

type fakeClient struct {
	objects map[string]fakeObject
	history map[string][]WriteObjectOptions
	nextGen int64
}

type fakeObject struct {
	data       []byte
	generation int64
}

func newFakeClient() *fakeClient {
	return &fakeClient{
		objects: make(map[string]fakeObject),
		history: make(map[string][]WriteObjectOptions),
		nextGen: 1,
	}
}

func (c *fakeClient) ReadObject(_ context.Context, bucket, object string) (ReadObjectResult, error) {
	entry, ok := c.objects[key(bucket, object)]
	if !ok {
		return ReadObjectResult{}, gcsapi.ErrObjectNotExist
	}

	return ReadObjectResult{
		Data:       slices.Clone(entry.data),
		Generation: entry.generation,
	}, nil
}

func (c *fakeClient) WriteObject(_ context.Context, bucket, object string, data []byte, opts WriteObjectOptions) (int64, error) {
	storeKey := key(bucket, object)
	entry, exists := c.objects[storeKey]
	c.history[object] = append(c.history[object], opts)

	switch {
	case opts.DoesNotExist && exists:
		return 0, &googleapi.Error{Code: 412}
	case opts.MatchGeneration > 0 && (!exists || entry.generation != opts.MatchGeneration):
		return 0, &googleapi.Error{Code: 412}
	}

	generation := c.nextGen
	c.nextGen++
	c.objects[storeKey] = fakeObject{
		data:       slices.Clone(data),
		generation: generation,
	}

	return generation, nil
}

func (c *fakeClient) lastWriteOptions(object string) WriteObjectOptions {
	history := c.history[object]
	if len(history) == 0 {
		return WriteObjectOptions{}
	}

	return history[len(history)-1]
}

func key(bucket, object string) string {
	return fmt.Sprintf("%s/%s", bucket, object)
}
