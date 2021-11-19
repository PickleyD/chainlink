package internal_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/onsi/gomega"
	"github.com/smartcontractkit/chainlink/core/internal/cltest"
	"github.com/smartcontractkit/chainlink/core/internal/cltest/heavyweight"
	"github.com/smartcontractkit/chainlink/core/internal/gethwrappers/generated/link_token_interface"
	"github.com/smartcontractkit/chainlink/core/internal/testutils/configtest"
	"github.com/smartcontractkit/chainlink/core/logger"
	"github.com/smartcontractkit/chainlink/core/services/keystore/keys/ocr2key"
	"github.com/smartcontractkit/chainlink/core/services/offchainreporting2"
	"github.com/smartcontractkit/chainlink/core/store/models"
	"github.com/smartcontractkit/libocr/commontypes"
	ocr2aggregator "github.com/smartcontractkit/libocr/gethwrappers2/ocr2aggregator"
	testoffchainaggregator2 "github.com/smartcontractkit/libocr/gethwrappers2/testocr2aggregator"
	confighelper2 "github.com/smartcontractkit/libocr/offchainreporting2/confighelper"
	ocrtypes2 "github.com/smartcontractkit/libocr/offchainreporting2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v4"
)

func setupOCR2Contracts(t *testing.T) (*bind.TransactOpts, *backends.SimulatedBackend, common.Address, *ocr2aggregator.OCR2Aggregator) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err, "failed to generate ethereum identity")
	owner := cltest.MustNewSimulatedBackendKeyedTransactor(t, key)
	sb := new(big.Int)
	sb, _ = sb.SetString("100000000000000000000", 10) // 1 eth
	genesisData := core.GenesisAlloc{owner.From: {Balance: sb}}
	gasLimit := ethconfig.Defaults.Miner.GasCeil * 2
	b := backends.NewSimulatedBackend(genesisData, gasLimit)
	linkTokenAddress, _, linkContract, err := link_token_interface.DeployLinkToken(owner, b)
	require.NoError(t, err)
	accessAddress, _, _, err := testoffchainaggregator2.DeploySimpleWriteAccessController(owner, b)
	require.NoError(t, err, "failed to deploy test access controller contract")
	b.Commit()

	minAnswer, maxAnswer := new(big.Int), new(big.Int)
	minAnswer.Exp(big.NewInt(-2), big.NewInt(191), nil)
	maxAnswer.Exp(big.NewInt(2), big.NewInt(191), nil)
	maxAnswer.Sub(maxAnswer, big.NewInt(1))
	ocrContractAddress, _, ocrContract, err := ocr2aggregator.DeployOCR2Aggregator(
		owner,
		b,
		linkTokenAddress, //_link common.Address,
		minAnswer,        // -2**191
		maxAnswer,        // 2**191 - 1
		accessAddress,
		accessAddress,
		9,
		"TEST",
	)

	require.NoError(t, err)
	_, err = linkContract.Transfer(owner, ocrContractAddress, big.NewInt(1000))
	require.NoError(t, err)
	b.Commit()
	return owner, b, ocrContractAddress, ocrContract
}

func setupNodeOCR2(t *testing.T, owner *bind.TransactOpts, port uint16, dbName string, b *backends.SimulatedBackend) (*cltest.TestApplication, string, common.Address, ocr2key.KeyBundle, *configtest.TestGeneralConfig, func()) {
	config, _, ormCleanup := heavyweight.FullTestORM(t, fmt.Sprintf("%s%d", dbName, port), true, true)
	config.Overrides.FeatureOffchainReporting2 = null.BoolFrom(true)

	app, appCleanup := cltest.NewApplicationWithConfigAndKeyOnSimulatedBlockchain(t, config, b)
	_, err := app.GetKeyStore().P2P().Create()
	require.NoError(t, err)
	p2pIDs, err := app.GetKeyStore().P2P().GetAll()
	require.NoError(t, err)
	require.Len(t, p2pIDs, 1)
	peerID := p2pIDs[0].PeerID()

	config.Overrides.OCR2P2PPeerID = peerID
	config.Overrides.OCR2P2PListenPort = port
	p2paddresses := []string{
		fmt.Sprintf("127.0.0.1:%d", port),
	}
	config.Overrides.OCR2P2PV2ListenAddresses = p2paddresses
	config.Overrides.OCR2P2PV2AnnounceAddresses = p2paddresses

	// Disables ocr spec validation so we can have fast polling for the test.
	config.Overrides.Dev = null.BoolFrom(true)

	sendingKeys, err := app.KeyStore.Eth().SendingKeys()
	require.NoError(t, err)
	require.Len(t, sendingKeys, 1)
	transmitter := sendingKeys[0].Address.Address()
	logger.Debug(fmt.Sprintf("Transmitter address %s", transmitter))

	// Fund the transmitter address with some ETH
	n, err := b.NonceAt(context.Background(), owner.From, nil)
	require.NoError(t, err)

	tx := types.NewTransaction(n, transmitter, big.NewInt(1000000000000000000), 21000, big.NewInt(1000000000), nil)
	signedTx, err := owner.Signer(owner.From, tx)
	require.NoError(t, err)
	err = b.SendTransaction(context.Background(), signedTx)
	require.NoError(t, err)
	b.Commit()

	kb, err := app.GetKeyStore().OCR2().Create()
	require.NoError(t, err)
	return app, peerID.Raw(), transmitter, kb, config, func() {
		ormCleanup()
		appCleanup()
	}
}

func TestIntegration_OCR2(t *testing.T) {
	owner, b, ocrContractAddress, ocrContract := setupOCR2Contracts(t)

	// Note it's plausible these ports could be occupied on a CI machine.
	// May need a port randomize + retry approach if we observe collisions.
	bootstrapNodePort := uint16(19999)
	appBootstrap, bootstrapPeerID, _, _, _, cleanup := setupNodeOCR2(t, owner, bootstrapNodePort, "bootstrap", b)
	defer cleanup()

	var (
		oracles      []confighelper2.OracleIdentityExtra
		transmitters []common.Address
		kbs          []ocr2key.KeyBundle
		apps         []*cltest.TestApplication
	)
	for i := uint16(0); i < 4; i++ {
		app, peerID, transmitter, kb, cfg, cleanup := setupNodeOCR2(t, owner, bootstrapNodePort+1+i, fmt.Sprintf("oracle%d", i), b)
		defer cleanup()
		// GracePeriod < ObservationTimeout
		cfg.Overrides.OCR2ObservationGracePeriod = 100 * time.Millisecond

		// Supply the bootstrap IP and port as a V2 peer address
		cfg.Overrides.OCR2P2PV2Bootstrappers = []commontypes.BootstrapperLocator{
			{PeerID: bootstrapPeerID, Addrs: []string{
				fmt.Sprintf("127.0.0.1:%d", bootstrapNodePort),
			}},
		}

		kbs = append(kbs, kb)
		apps = append(apps, app)
		transmitters = append(transmitters, transmitter)

		oracles = append(oracles, confighelper2.OracleIdentityExtra{
			OracleIdentity: confighelper2.OracleIdentity{
				OnChainSigningAddress: kb.OnchainKeyring.SigningAddress().Bytes(),
				TransmitAccount:       ocrtypes2.Account(transmitter.String()),
				OffchainPublicKey:     kb.OffchainKeyring.OffchainPublicKey(),
				PeerID:                peerID,
			},
			ConfigEncryptionPublicKey: kb.OffchainKeyring.ConfigEncryptionPublicKey(),
		})
	}

	tick := time.NewTicker(1 * time.Second)
	defer tick.Stop()
	go func() {
		for range tick.C {
			b.Commit()
		}
	}()

	logger.Debugw("Setting Payees on Oracle Contract", "transmitters", transmitters)
	_, err := ocrContract.SetPayees(
		owner,
		transmitters,
		transmitters,
	)
	require.NoError(t, err)
	signers, transmitters, threshold, onchainConfig, encodedConfigVersion, encodedConfig, err := confighelper2.ContractSetConfigArgsForIntegrationTest(
		oracles,
		1,
		1000000000/100, // threshold PPB
	)
	require.NoError(t, err)
	logger.Debugw("Setting Config on Oracle Contract",
		"signers", signers,
		"transmitters", transmitters,
		"threshold", threshold,
		"onchainConfig", onchainConfig,
		"encodedConfigVersion", encodedConfigVersion,
	)
	_, err = ocrContract.SetConfig(
		owner,
		signers,
		transmitters,
		threshold,
		onchainConfig,
		encodedConfigVersion,
		encodedConfig,
	)
	require.NoError(t, err)
	b.Commit()

	err = appBootstrap.Start()
	require.NoError(t, err)
	defer appBootstrap.Stop()

	ocrJob, err := offchainreporting2.ValidatedOracleSpecToml(appBootstrap.GetChainSet(), fmt.Sprintf(`
type               = "offchainreporting2"
schemaVersion      = 1
name               = "boot"
contractAddress    = "%s"
isBootstrapPeer    = true
`, ocrContractAddress))
	require.NoError(t, err)
	_, err = appBootstrap.AddJobV2(context.Background(), ocrJob, null.NewString("boot", true))
	require.NoError(t, err)

	var jids []int32
	var servers, slowServers = make([]*httptest.Server, 4), make([]*httptest.Server, 4)
	// We expect metadata of:
	//  latestAnswer:nil // First call
	//  latestAnswer:0
	//  latestAnswer:10
	//  latestAnswer:20
	//  latestAnswer:30
	var metaLock sync.Mutex
	expectedMeta := map[string]struct{}{
		"0": {}, "10": {}, "20": {}, "30": {},
	}
	for i := 0; i < 4; i++ {
		err = apps[i].Start()
		require.NoError(t, err)
		defer apps[i].Stop()

		// Since this API speed is > ObservationTimeout we should ignore it and still produce values.
		slowServers[i] = httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			time.Sleep(5 * time.Second)
			res.WriteHeader(http.StatusOK)
			res.Write([]byte(`{"data":10}`))
		}))
		defer slowServers[i].Close()
		servers[i] = httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			b, err := ioutil.ReadAll(req.Body)
			require.NoError(t, err)
			var m models.BridgeMetaDataJSON
			require.NoError(t, json.Unmarshal(b, &m))
			if m.Meta.LatestAnswer != nil && m.Meta.UpdatedAt != nil {
				metaLock.Lock()
				delete(expectedMeta, m.Meta.LatestAnswer.String())
				metaLock.Unlock()
			}
			res.WriteHeader(http.StatusOK)
			res.Write([]byte(`{"data":10}`))
		}))
		defer servers[i].Close()
		u, _ := url.Parse(servers[i].URL)
		apps[i].Store.CreateBridgeType(&models.BridgeType{
			Name: models.TaskType(fmt.Sprintf("bridge%d", i)),
			URL:  models.WebURL(*u),
		})

		// Note we need: observationTimeout + observationGracePeriod + DeltaGrace (500ms) < DeltaRound (1s)
		// So 200ms + 200ms + 500ms < 1s
		ocrJob, err := offchainreporting2.ValidatedOracleSpecToml(apps[i].GetChainSet(), fmt.Sprintf(`
type               = "offchainreporting2"
schemaVersion      = 1
name               = "web oracle spec"
contractAddress    = "%s"
isBootstrapPeer    = false
keyBundleID        = "%s"
transmitterAddress = "%s"
observationTimeout = "100ms"
contractConfigConfirmations = 1
contractConfigTrackerPollInterval = "1s"
observationSource = """
    // data source 1
    ds1          [type=bridge name="%s"];
    ds1_parse    [type=jsonparse path="data"];
    ds1_multiply [type=multiply times=%d];

    // data source 2
    ds2          [type=http method=GET url="%s"];
    ds2_parse    [type=jsonparse path="data"];
    ds2_multiply [type=multiply times=%d];

    ds1 -> ds1_parse -> ds1_multiply -> answer1;
    ds2 -> ds2_parse -> ds2_multiply -> answer1;

	answer1 [type=median index=0];
"""
`, ocrContractAddress, kbs[i].ID(), transmitters[i], fmt.Sprintf("bridge%d", i), i, slowServers[i].URL, i))
		require.NoError(t, err)
		jb, err := apps[i].AddJobV2(context.Background(), ocrJob, null.NewString("testocr", true))
		require.NoError(t, err)
		jids = append(jids, jb.ID)
	}

	// Assert that all the OCR jobs get a run with valid values eventually.
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		ic := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Want at least 2 runs so we see all the metadata.
			pr := cltest.WaitForPipelineComplete(t, ic, jids[ic], 2, 0, apps[ic].JobORM(), 1*time.Minute, 1*time.Second)
			jb, err := pr[0].Outputs.MarshalJSON()
			require.NoError(t, err)
			assert.Equal(t, []byte(fmt.Sprintf("[\"%d\"]", 10*ic)), jb)
			require.NoError(t, err)
		}()
	}
	wg.Wait()

	// 4 oracles reporting 0, 10, 20, 30. Answer should be 20 (results[4/2]).
	gomega.NewGomegaWithT(t).Eventually(func() string {
		answer, err := ocrContract.LatestAnswer(nil)
		require.NoError(t, err)
		return answer.String()
	}, 30*time.Second, 200*time.Millisecond).Should(gomega.Equal("20"))

	for _, app := range apps {
		jobs, _, err := app.JobORM().JobsV2(0, 1000)
		require.NoError(t, err)
		// No spec errors
		for _, j := range jobs {
			ignore := 0
			for i := range j.JobSpecErrors {
				// Non-fatal timing related error, ignore for testing.
				if strings.Contains(j.JobSpecErrors[i].Description, "leader's phase conflicts tGrace timeout") {
					ignore++
				}
			}
			require.Len(t, j.JobSpecErrors, ignore)
		}
	}
	assert.Len(t, expectedMeta, 0, "expected metadata %v", expectedMeta)
}