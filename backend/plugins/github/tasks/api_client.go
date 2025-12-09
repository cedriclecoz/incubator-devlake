/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tasks

import (
	"net/http"
	"strconv"
	"time"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/github/models"
)

func CreateApiClient(taskCtx plugin.TaskContext, connection *models.GithubConnection) (*api.ApiAsyncClient, errors.Error) {
	apiClient, err := api.NewApiClientFromConnection(taskCtx.GetContext(), taskCtx, connection)
	if err != nil {
		return nil, err
	}

	logger := taskCtx.GetLogger()

	// Create a holder for the async client that can be captured by closure before it's populated
	type asyncClientHolder struct {
		client            *api.ApiAsyncClient
		pendingAdjustment *time.Duration // Store adjustment if client not ready yet
	}
	holder := &asyncClientHolder{}

	// Set up dynamic rate limit adjustment callback BEFORE creating async client
	logger.Info("CLCLCLC setting up dynamic rate limit adjustment for GitHub API")
	apiClient.SetAfterFunction(func(res *http.Response) errors.Error {
		// Extract rate limit information from headers
		// GitHub uses lowercase header names: x-ratelimit-remaining, x-ratelimit-reset
		remainingStr := res.Header.Get("X-RateLimit-Remaining")
		resetStr := res.Header.Get("X-RateLimit-Reset")
		dateStr := res.Header.Get("Date")

		// Debug: log what we got
		if remainingStr != "" || resetStr != "" {
			logger.Info("CLCLCLC found rate limit headers: remaining=%q, reset=%q", remainingStr, resetStr)
		}

		if remainingStr == "" || resetStr == "" || dateStr == "" {
			// Headers not available, skip adjustment
			return nil
		}

		logger.Info("CLCLCLC rate limit headers: remaining=%q, reset=%q, date=%q", remainingStr, resetStr, dateStr)

		remaining, err := strconv.Atoi(remainingStr)
		if err != nil {
			logger.Debug("CLCLCLC failed to parse remaining: %v", err)
			return nil
		}

		resetInt, err := strconv.ParseInt(resetStr, 10, 64)
		if err != nil {
			logger.Debug("CLCLCLC failed to parse reset: %v", err)
			return nil
		}
		resetTime := time.Unix(resetInt, 0)

		date, err := http.ParseTime(dateStr)
		if err != nil {
			logger.Debug("CLCLCLC failed to parse date: %v", err)
			return nil
		}

		// Calculate time until reset
		timeUntilReset := resetTime.Sub(date)
		if timeUntilReset <= 0 {
			// Reset time has passed, rate limit should refresh soon
			return nil
		}

		// Calculate required requests per second based on remaining quota
		// Apply safety multiplier to avoid hitting limit exactly
		adjustedRemaining := float64(remaining) * 0.95
		requestsPerSecond := adjustedRemaining / timeUntilReset.Seconds()

		// Calculate new tick interval and adjust if the holder has been populated
		if requestsPerSecond > 0 {
			newTickInterval := time.Duration(float64(time.Second) / requestsPerSecond)

			if holder.client != nil {
				currentInterval := holder.client.GetTickInterval()
				logger.Info("CLCLCLC current=%s, new=%s, remaining=%d, reset_in=%.0fs", currentInterval, newTickInterval, remaining, timeUntilReset.Seconds())
				// Only adjust if the change is significant (>10% difference)
				if newTickInterval > currentInterval*11/10 || newTickInterval < currentInterval*9/10 {
					logger.Info(
						"CLCLCLC adjusting rate limit: remaining=%d, reset_in=%.0fs, new_interval=%s (was %s)",
						remaining,
						timeUntilReset.Seconds(),
						newTickInterval,
						currentInterval,
					)
					holder.client.Reset(newTickInterval)
				}
			} else {
				// Client not ready yet, store the adjustment for later
				logger.Info("CLCLCLC holder.client is nil, storing pending adjustment: %s", newTickInterval)
				holder.pendingAdjustment = &newTickInterval
			}
		}

		return nil
	})

	// create rate limit calculator
	rateLimiter := &api.ApiRateLimitCalculator{
		UserRateLimitPerHour: connection.RateLimitPerHour,
		Method:               http.MethodGet,
		DynamicRateLimit: func(res *http.Response) (int, time.Duration, errors.Error) {
			// Use static limit for initial setup only
			var rateLimit int
			headerRateLimit := res.Header.Get("X-RateLimit-Limit")
			if len(headerRateLimit) > 0 {
				var e error
				rateLimit, e = strconv.Atoi(headerRateLimit)
				if e != nil {
					return 0, 0, errors.Default.Wrap(err, "failed to parse X-RateLimit-Limit header")
				}
			} else {
				// if we can't find "X-RateLimit-Limit" in header, we will return globalRatelimit in ApiRateLimitCalculator.Calculate
				return 0, 0, nil
			}
			return rateLimit * connection.GetTokensCount(), 1 * time.Hour, nil
		},
	}
	asyncApiClient, err := api.CreateAsyncApiClient(
		taskCtx,
		apiClient,
		rateLimiter,
	)
	if err != nil {
		return nil, err
	}

	// Populate the holder so the callback can now access the asyncApiClient
	holder.client = asyncApiClient

	// Apply any pending adjustment that was calculated before the client was ready
	if holder.pendingAdjustment != nil {
		logger.Info("CLCLCLC applying pending rate limit adjustment: %s", *holder.pendingAdjustment)
		asyncApiClient.Reset(*holder.pendingAdjustment)
	}

	return asyncApiClient, nil
}
