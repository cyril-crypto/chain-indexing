package tmcosmosutils_test

import (
	"encoding/base64"

	"github.com/crypto-com/chain-indexing/internal/tmcosmosutils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const tendermintPubKey = "na51D8RmKXyWrid9I6wtdxgP6f1Nl3EyNNEzqxVquoM="
const tendermintAddress = "B5EC6D86F8F418F480799447F5C21F1C17C6F8F8"
const consensusNodeAddress = "tcrocnclcons1khkxmphc7sv0fqrej3rltsslrstud78cam9ekl"
const consensusNodePubKey = "tcrocnclconspub1zcjduepqnkh82r7yvc5he94wya7j8tpdwuvql60afkthzv356ye6k9t2h2psr3u067"

var _ = Describe("tmcosmosutils", func() {
	Describe("TmAddressFromTmPubKey", func() {
		It("should work", func() {
			pubKey, _ := base64.StdEncoding.DecodeString(tendermintPubKey)
			Expect(tmcosmosutils.TmAddressFromTmPubKey(
				pubKey,
			)).To(Equal(tendermintAddress))
		})
	})

	Describe("ConsensusNodeAddressFromTmPubKey", func() {
		It("should work", func() {
			pubKey, _ := base64.StdEncoding.DecodeString(tendermintPubKey)
			Expect(tmcosmosutils.ConsensusNodeAddressFromTmPubKey(
				"tcrocnclcons", pubKey,
			)).To(Equal(consensusNodeAddress))
		})
	})

	Describe("ConsensusNodePubKeyFromTmPubKey", func() {
		It("should work", func() {
			pubKey, _ := base64.StdEncoding.DecodeString(tendermintPubKey)
			Expect(tmcosmosutils.ConsensusNodePubKeyFromTmPubKey(
				"tcrocnclconspub", pubKey,
			)).To(Equal(consensusNodePubKey))
		})
	})

	Describe("ConsensusNodeAddressFromConsensusNodePubKey", func() {
		It("should work", func() {
			Expect(tmcosmosutils.ConsensusNodeAddressFromConsensusNodePubKey(
				"tcrocnclcons", consensusNodePubKey,
			)).To(Equal(consensusNodeAddress))
		})

	})

	Describe("AddressFromPubKey", func() {
		It("should work", func() {
			pubkey, _ := base64.StdEncoding.DecodeString("A3ill3YNyWvcMstrbssC9SpzhMm+tCMWPB7bgOqWQZYk")
			Expect(tmcosmosutils.AccountAddressFromPubKey(
				"tcro", pubkey,
			)).To(Equal("tcro1p4fzn6ta24c6ek4v2qls6y5uug44ku9tnypcaf"))
		})
	})
})
