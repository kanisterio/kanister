package gcepd

import (
	"context"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
)

// Client is a wrapper for Client client
type Client struct {
	Service   *compute.Service
	ProjectID string
}

// NewClient returns a Client struct
func NewClient(ctx context.Context, servicekey string) (*Client, error) {
	var err error
	var creds *google.Credentials
	if len(servicekey) > 0 {
		creds, err = google.CredentialsFromJSON(ctx, []byte(servicekey), compute.ComputeScope)
	} else {
		creds, err = google.FindDefaultCredentials(ctx, compute.ComputeScope)
	}

	if err != nil {
		return nil, err
	}
	// NOTE: Ashlie is not sure how long this will work for since a comment at
	// https://godoc.org/golang.org/x/oauth2#NewClient
	// states that the client in the context will only be used for getting a
	// token.
	httpClient := &http.Client{Transport: http.DefaultTransport}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)

	client := oauth2.NewClient(ctx, creds.TokenSource)

	service, err := compute.New(client)
	if err != nil {
		return nil, err
	}

	return &Client{
		Service:   service,
		ProjectID: creds.ProjectID,
	}, nil
}
