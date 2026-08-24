// SPDX-License-Identifier: MIT
//go:build integration

package e2e

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MCP process lifecycle", func() {
	It("rejects non-JSON and non-JSON-RPC stdout", func() {
		Expect(validateJSONRPCFrames([]byte("diagnostic noise\n"))).To(MatchError(ContainSubstring("not JSON")))
		Expect(validateJSONRPCFrames([]byte(`{"result":{}}` + "\n"))).To(MatchError(ContainSubstring("jsonrpc")))
		Expect(validateJSONRPCFrames([]byte(`{"jsonrpc":"2.0","id":1,"result":{}}` + "\n"))).To(Succeed())
	})

	It("owns one bounded session and closes idempotently on EOF", func() {
		workspace, _, err := materializeCanonical(context.Background(), GinkgoT().TempDir(), SeedMissingOnly)
		Expect(err).NotTo(HaveOccurred())
		started := time.Now()
		session, err := startMCPSession(context.Background(), workspace)
		Expect(err).NotTo(HaveOccurred())
		Expect(session.Close()).To(Succeed())
		Expect(session.Close()).To(Succeed())
		Expect(time.Since(started)).To(BeNumerically("<", maximumScenarioTimeout))
		Expect(len(session.stdout.Bytes())).To(BeNumerically("<", maximumCapturedOutput+64))
		Expect(len(session.stderr.Bytes())).To(BeNumerically("<", maximumCapturedOutput+64))
	})
})
