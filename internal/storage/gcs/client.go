package gcs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"

	gcsapi "cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"

	"github.com/nkoji21/leetdaily/internal/storage"
)

type GoogleClient struct {
	client *gcsapi.Client
}

func NewGoogleClient(ctx context.Context) (ObjectClient, error) {
	client, err := gcsapi.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	return &GoogleClient{client: client}, nil
}

func (c *GoogleClient) ReadObject(ctx context.Context, bucket, object string) (ReadObjectResult, error) {
	reader, err := c.client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return ReadObjectResult{}, err
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return ReadObjectResult{}, fmt.Errorf("read gs://%s/%s: %w", bucket, object, err)
	}

	return ReadObjectResult{
		Data:       data,
		Generation: reader.Attrs.Generation,
	}, nil
}

func (c *GoogleClient) WriteObject(ctx context.Context, bucket, object string, data []byte, opts WriteObjectOptions) (int64, error) {
	handle := c.client.Bucket(bucket).Object(object)
	if opts.DoesNotExist {
		handle = handle.If(gcsapi.Conditions{DoesNotExist: true})
	} else if opts.MatchGeneration > 0 {
		handle = handle.If(gcsapi.Conditions{GenerationMatch: opts.MatchGeneration})
	}

	writer := handle.NewWriter(ctx)
	writer.ContentType = "application/json"

	if _, err := writer.Write(data); err != nil {
		_ = writer.Close()
		return 0, fmt.Errorf("write gs://%s/%s: %w", bucket, object, err)
	}

	if err := writer.Close(); err != nil {
		return 0, err
	}

	attrs := writer.Attrs()
	if attrs == nil {
		return 0, fmt.Errorf("write gs://%s/%s: missing object attributes after close", bucket, object)
	}

	return attrs.Generation, nil
}

func normalizeError(err error, object string) error {
	switch {
	case errors.Is(err, gcsapi.ErrObjectNotExist):
		return fmt.Errorf("%w: %s", storage.ErrNotFound, object)
	case isPreconditionFailed(err):
		return fmt.Errorf("%w: %s", storage.ErrConflict, object)
	default:
		return err
	}
}

func writeOptionsFromVersion(version storage.Version) WriteObjectOptions {
	if version.IsZero() {
		return WriteObjectOptions{DoesNotExist: true}
	}

	generation, err := strconv.ParseInt(version.Token, 10, 64)
	if err != nil {
		return WriteObjectOptions{DoesNotExist: true}
	}

	return WriteObjectOptions{MatchGeneration: generation}
}

func versionFromGeneration(generation int64) storage.Version {
	if generation < 1 {
		return storage.Version{}
	}

	return storage.Version{Token: strconv.FormatInt(generation, 10)}
}

func isPreconditionFailed(err error) bool {
	var googleErr *googleapi.Error
	return errors.As(err, &googleErr) && googleErr.Code == 412
}
