// Copyright 2019 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gcepd

import (
	"context"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
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

	service, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	return &Client{
		Service:   service,
		ProjectID: creds.ProjectID,
	}, nil
}
