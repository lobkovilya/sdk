// Copyright (c) 2020 Cisco and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package authorize_test

import (
	"context"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/open-policy-agent/opa/rego"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"testing"
)

func TestAuthzEndpoint(t *testing.T) {
	//r := require.New(t)

	suits := []struct {
		name     string
		policy   string
		request  *networkservice.NetworkServiceRequest
		response *connection.Connection
		denied   bool
	}{
		{
			name: "simple positive test",
			policy: `
			package test
		
			default allow = false
		
			allow {
				input = "allowed"
			}
			`,
			request: &networkservice.NetworkServiceRequest{
				Connection: &connection.Connection{
					Id: "conn-1",
					Path: &connection.Path{
						Index: 0,
						PathSegments: []*connection.PathSegment{
							{
								Token: "allowed",
							},
						},
					},
				},
			},
			denied: false,
		},
		{
			name: "simple negative test",
			policy: `
			package test
		
			default allow = false
		
			allow {
				input = "allowed"
			}
			`,
			request: &networkservice.NetworkServiceRequest{
				Connection: &connection.Connection{
					Id: "conn-1",
					Path: &connection.Path{
						Index: 0,
						PathSegments: []*connection.PathSegment{
							{
								Token: "not_allowed",
							},
						},
					},
				},
			},
			denied: true,
		},
	}

	for i := range suits {
		s := suits[i]
		t.Run(s.name, func(t *testing.T) {
			p, err := rego.New(
				rego.Query("data.test.allow"),
				rego.Module("example.com", s.policy)).PrepareForEval(context.Background())
			require.Nilf(t, err, "failed to create new rego policy: %v", err)

			srv := chain.NewNetworkServiceServer(authorize.NewServer(p))

			checkResult := func(err error) {
				if !s.denied {
					require.Nil(t, err, "request expected to be not denied")
					return
				}

				require.NotNil(t, err, "request expected to be denied")
				s, ok := status.FromError(err)
				require.True(t, ok, "error without error status code")
				require.Equal(t, s.Code(), codes.PermissionDenied, "wrong error status code")
			}

			_, err = srv.Request(context.Background(), s.request)
			checkResult(err)

			_, err = srv.Close(context.Background(), s.request.GetConnection())
			checkResult(err)
		})
	}
}