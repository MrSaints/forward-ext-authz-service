// Copyright Project Contour Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Modified from https://github.com/projectcontour/contour-authserver/blob/f88f17864d16b053e1387a8778a13bfcb511a5e5/pkg/auth/server.go

package main

import (
	"context"

	"go.uber.org/zap"

	envoy_service_auth_v3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
)

type CheckRequestV3 = envoy_service_auth_v3.CheckRequest
type CheckResponseV3 = envoy_service_auth_v3.CheckResponse

type authV3 struct {
	logger  *zap.Logger
	checker checker
}

func (a *authV3) Check(ctx context.Context, check *CheckRequestV3) (*CheckResponseV3, error) {
	a.logger.Debug("Handling v3 request", zap.Any("check", check))

	req := Request{}
	_, err := req.FromV3(check)
	if err != nil {
		return nil, err
	}

	res, err := a.checker.Check(ctx, &req)
	if err != nil {
		return nil, err
	}

	return res.AsV3(), nil
}
