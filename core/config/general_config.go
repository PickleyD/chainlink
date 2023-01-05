package config

import (
	"math/big"
	"net/url"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/spf13/viper"
	"go.uber.org/zap/zapcore"

	"github.com/pickleyd/chainlink/core/assets"
	"github.com/pickleyd/chainlink/core/config/envvar"
	"github.com/pickleyd/chainlink/core/logger"
	"github.com/pickleyd/chainlink/core/logger/audit"
	"github.com/pickleyd/chainlink/core/store/dialects"
	"github.com/pickleyd/chainlink/core/store/models"
	"github.com/pickleyd/chainlink/core/utils"
)

//go:generate mockery --quiet --name GeneralConfig --output ./mocks/ --case=underscore

//nolint
var (
	ErrUnset   = errors.New("env var unset")
	ErrInvalid = errors.New("env var invalid")

	configFileNotFoundError = reflect.TypeOf(viper.ConfigFileNotFoundError{})
)

type GeneralOnlyConfig interface {
	// Validate() error
	// SetLogLevel(lvl zapcore.Level) error
	// SetLogSQL(logSQL bool)

	AutoPprofEnabled() bool
	EVMEnabled() bool
	EVMRPCEnabled() bool
	KeeperCheckUpkeepGasPriceFeatureEnabled() bool
	P2PEnabled() bool
	SolanaEnabled() bool
	TerraEnabled() bool
	StarkNetEnabled() bool
}

type LogFn func(...any)

type BasicConfig interface {
	Validate() error
	LogConfiguration(log LogFn)
	SetLogLevel(lvl zapcore.Level) error
	SetLogSQL(logSQL bool)
	SetPasswords(keystore, vrf *string)

	FeatureFlags
	audit.Config

	AdvisoryLockCheckInterval() time.Duration
	AdvisoryLockID() int64
	AllowOrigins() string
	AppID() uuid.UUID
	AuthenticatedRateLimit() int64
	AuthenticatedRateLimitPeriod() models.Duration
	AutoPprofBlockProfileRate() int
	AutoPprofCPUProfileRate() int
	AutoPprofGatherDuration() models.Duration
	AutoPprofGatherTraceDuration() models.Duration
	AutoPprofGoroutineThreshold() int
	AutoPprofMaxProfileSize() utils.FileSize
	AutoPprofMemProfileRate() int
	AutoPprofMemThreshold() utils.FileSize
	AutoPprofMutexProfileFraction() int
	AutoPprofPollInterval() models.Duration
	AutoPprofProfileRoot() string
	BlockBackfillDepth() uint64
	BlockBackfillSkip() bool
	BridgeResponseURL() *url.URL
	BridgeCacheTTL() time.Duration
	CertFile() string
	DatabaseBackupDir() string
	DatabaseBackupFrequency() time.Duration
	DatabaseBackupMode() DatabaseBackupMode
	DatabaseBackupOnVersionUpgrade() bool
	DatabaseBackupURL() *url.URL
	DatabaseDefaultIdleInTxSessionTimeout() time.Duration
	DatabaseDefaultLockTimeout() time.Duration
	DatabaseDefaultQueryTimeout() time.Duration
	DatabaseListenerMaxReconnectDuration() time.Duration
	DatabaseListenerMinReconnectInterval() time.Duration
	DatabaseLockingMode() string
	DatabaseURL() url.URL
	DefaultChainID() *big.Int
	DefaultHTTPLimit() int64
	DefaultHTTPTimeout() models.Duration
	DefaultLogLevel() zapcore.Level
	Dev() bool
	ShutdownGracePeriod() time.Duration
	EthereumHTTPURL() *url.URL
	EthereumNodes() string
	EthereumSecondaryURLs() []url.URL
	EthereumURL() string
	ExplorerAccessKey() string
	ExplorerSecret() string
	ExplorerURL() *url.URL
	FMDefaultTransactionQueueDepth() uint32
	FMSimulateTransactions() bool
	GetAdvisoryLockIDConfiguredOrDefault() int64
	GetDatabaseDialectConfiguredOrDefault() dialects.DialectName
	HTTPServerWriteTimeout() time.Duration
	InsecureFastScrypt() bool
	JSONConsole() bool
	JobPipelineMaxRunDuration() time.Duration
	JobPipelineReaperInterval() time.Duration
	JobPipelineReaperThreshold() time.Duration
	JobPipelineResultWriteQueueDepth() uint64
	KeeperDefaultTransactionQueueDepth() uint32
	KeeperGasPriceBufferPercent() uint16
	KeeperGasTipCapBufferPercent() uint16
	KeeperBaseFeeBufferPercent() uint16
	KeeperMaximumGracePeriod() int64
	KeeperRegistryCheckGasOverhead() uint32
	KeeperRegistryPerformGasOverhead() uint32
	KeeperRegistryMaxPerformDataSize() uint32
	KeeperRegistrySyncInterval() time.Duration
	KeeperRegistrySyncUpkeepQueueSize() uint32
	KeeperTurnLookBack() int64
	KeeperTurnFlagEnabled() bool
	KeyFile() string
	KeystorePassword() string
	LeaseLockDuration() time.Duration
	LeaseLockRefreshInterval() time.Duration
	LogFileDir() string
	LogLevel() zapcore.Level
	LogSQL() bool
	LogFileMaxSize() utils.FileSize
	LogFileMaxAge() int64
	LogFileMaxBackups() int64
	LogUnixTimestamps() bool
	MercuryCredentials(url string) (username, password string, err error)
	MigrateDatabase() bool
	ORMMaxIdleConns() int
	ORMMaxOpenConns() int
	Port() uint16
	PyroscopeAuthToken() string
	PyroscopeServerAddress() string
	PyroscopeEnvironment() string
	RPID() string
	RPOrigin() string
	ReaperExpiration() models.Duration
	RootDir() string
	SecureCookies() bool
	SentryDSN() string
	SentryDebug() bool
	SentryEnvironment() string
	SentryRelease() string
	SessionOptions() sessions.Options
	SessionTimeout() models.Duration
	SolanaNodes() string
	StarkNetNodes() string
	TerraNodes() string
	TLSCertPath() string
	TLSDir() string
	TLSHost() string
	TLSKeyPath() string
	TLSPort() uint16
	TLSRedirect() bool
	TelemetryIngressLogging() bool
	TelemetryIngressUniConn() bool
	TelemetryIngressServerPubKey() string
	TelemetryIngressURL() *url.URL
	TelemetryIngressBufferSize() uint
	TelemetryIngressMaxBatchSize() uint
	TelemetryIngressSendInterval() time.Duration
	TelemetryIngressSendTimeout() time.Duration
	TelemetryIngressUseBatchSend() bool
	TriggerFallbackDBPollInterval() time.Duration
	UnAuthenticatedRateLimit() int64
	UnAuthenticatedRateLimitPeriod() models.Duration
	VRFPassword() string

	OCR1Config
	OCR2Config

	P2PNetworking
	P2PV1Networking
	P2PV2Networking
}

// GlobalConfig holds global ENV overrides for EVM chains
// If set the global ENV will override everything
// The second bool indicates if it is set or not
type GlobalConfig interface {
	GlobalBalanceMonitorEnabled() (bool, bool)
	GlobalBlockEmissionIdleWarningThreshold() (time.Duration, bool)
	GlobalBlockHistoryEstimatorBatchSize() (uint32, bool)
	GlobalBlockHistoryEstimatorBlockDelay() (uint16, bool)
	GlobalBlockHistoryEstimatorBlockHistorySize() (uint16, bool)
	GlobalBlockHistoryEstimatorEIP1559FeeCapBufferBlocks() (uint16, bool)
	GlobalBlockHistoryEstimatorCheckInclusionBlocks() (uint16, bool)
	GlobalBlockHistoryEstimatorCheckInclusionPercentile() (uint16, bool)
	GlobalBlockHistoryEstimatorTransactionPercentile() (uint16, bool)
	GlobalChainType() (string, bool)
	GlobalEthTxReaperInterval() (time.Duration, bool)
	GlobalEthTxReaperThreshold() (time.Duration, bool)
	GlobalEthTxResendAfterThreshold() (time.Duration, bool)
	GlobalEvmEIP1559DynamicFees() (bool, bool)
	GlobalEvmFinalityDepth() (uint32, bool)
	GlobalEvmGasBumpPercent() (uint16, bool)
	GlobalEvmGasBumpThreshold() (uint64, bool)
	GlobalEvmGasBumpTxDepth() (uint16, bool)
	GlobalEvmGasBumpWei() (*assets.Wei, bool)
	GlobalEvmGasFeeCapDefault() (*assets.Wei, bool)
	GlobalEvmGasLimitDefault() (uint32, bool)
	GlobalEvmGasLimitMax() (uint32, bool)
	GlobalEvmGasLimitMultiplier() (float32, bool)
	GlobalEvmGasLimitTransfer() (uint32, bool)
	GlobalEvmGasLimitOCRJobType() (uint32, bool)
	GlobalEvmGasLimitDRJobType() (uint32, bool)
	GlobalEvmGasLimitVRFJobType() (uint32, bool)
	GlobalEvmGasLimitFMJobType() (uint32, bool)
	GlobalEvmGasLimitKeeperJobType() (uint32, bool)
	GlobalEvmGasPriceDefault() (*assets.Wei, bool)
	GlobalEvmGasTipCapDefault() (*assets.Wei, bool)
	GlobalEvmGasTipCapMinimum() (*assets.Wei, bool)
	GlobalEvmHeadTrackerHistoryDepth() (uint32, bool)
	GlobalEvmHeadTrackerMaxBufferSize() (uint32, bool)
	GlobalEvmHeadTrackerSamplingInterval() (time.Duration, bool)
	GlobalEvmLogBackfillBatchSize() (uint32, bool)
	GlobalEvmLogPollInterval() (time.Duration, bool)
	GlobalEvmLogKeepBlocksDepth() (uint32, bool)
	GlobalEvmMaxGasPriceWei() (*assets.Wei, bool)
	GlobalEvmMaxInFlightTransactions() (uint32, bool)
	GlobalEvmMaxQueuedTransactions() (uint64, bool)
	GlobalEvmMinGasPriceWei() (*assets.Wei, bool)
	GlobalEvmNonceAutoSync() (bool, bool)
	GlobalEvmUseForwarders() (bool, bool)
	GlobalEvmRPCDefaultBatchSize() (uint32, bool)
	GlobalFlagsContractAddress() (string, bool)
	GlobalGasEstimatorMode() (string, bool)
	GlobalLinkContractAddress() (string, bool)
	GlobalOCRContractConfirmations() (uint16, bool)
	GlobalOCRContractTransmitterTransmitTimeout() (time.Duration, bool)
	GlobalOCRDatabaseTimeout() (time.Duration, bool)
	GlobalOCRObservationGracePeriod() (time.Duration, bool)
	GlobalOCR2AutomationGasLimit() (uint32, bool)
	GlobalOperatorFactoryAddress() (string, bool)
	GlobalMinIncomingConfirmations() (uint32, bool)
	GlobalMinimumContractPayment() (*assets.Link, bool)
	GlobalNodeNoNewHeadsThreshold() (time.Duration, bool)
	GlobalNodePollFailureThreshold() (uint32, bool)
	GlobalNodePollInterval() (time.Duration, bool)
	GlobalNodeSelectionMode() (string, bool)
	GlobalNodeSyncThreshold() (uint32, bool)
}

type GeneralConfig interface {
	GeneralOnlyConfig
	// GlobalConfig
}

// generalConfig holds parameters used by the application which can be overridden by
// setting environment variables.
//
// If you add an entry here which does not contain sensitive information, you
// should also update presenters.ConfigWhitelist and cmd_test.TestClient_RunNodeShowsEnv.
type generalConfig struct {
	lggr             logger.Logger
	viper            *viper.Viper
	randomP2PPort    uint16
	randomP2PPortMtx sync.RWMutex
	dialect          string
	advisoryLockID   int64
	logLevel         zapcore.Level
	defaultLogLevel  zapcore.Level
	logSQL           bool
	logMutex         sync.RWMutex
	genAppID         sync.Once
	appID            uuid.UUID

	passwordKeystore, passwordVRF string
	passwordMu                    sync.RWMutex // passwords are set after initialization
}

// NewGeneralConfig returns the config with the environment variables set to their
// respective fields, or their defaults if environment variables are not set.
func NewGeneralConfig(lggr logger.Logger) GeneralConfig {
	v := viper.New()
	c := newGeneralConfigWithViper(v, lggr.Named("GeneralConfig"))
	c.dialect = "pgx"
	return c
}

func newGeneralConfigWithViper(v *viper.Viper, lggr logger.Logger) (config *generalConfig) {
	schemaT := reflect.TypeOf(envvar.ConfigSchema{})
	for index := 0; index < schemaT.NumField(); index++ {
		item := schemaT.FieldByIndex([]int{index})
		name := item.Tag.Get("env")
		def, exists := item.Tag.Lookup("default")
		if exists {
			v.SetDefault(name, def)
		}
		_ = v.BindEnv(name, name)
	}

	config = &generalConfig{
		lggr:  lggr,
		viper: v,
	}

	if err := utils.EnsureDirAndMaxPerms(config.RootDir(), os.FileMode(0700)); err != nil {
		lggr.Fatalf(`Error creating root directory "%s": %+v`, config.RootDir(), err)
	}

	v.SetConfigName("chainlink")
	v.AddConfigPath(config.RootDir())
	err := v.ReadInConfig()
	if err != nil && reflect.TypeOf(err) != configFileNotFoundError {
		lggr.Warnf("Unable to load config file: %v\n", err)
	}

	ll, invalid := envvar.LogLevel.Parse()
	if invalid != "" {
		lggr.Error(invalid)
	}
	config.defaultLogLevel = ll

	config.logLevel = config.defaultLogLevel
	config.logSQL = viper.GetBool(envvar.Name("LogSQL"))

	return
}

// RootDir represents the location on the file system where Chainlink should
// keep its files.
func (c *generalConfig) RootDir() string {
	return getEnvWithFallback(c, envvar.RootDir)
}

func getEnvWithFallback[T any](c *generalConfig, e *envvar.EnvVar[T]) T {
	v, invalid, err := e.ParseFrom(c.viper.GetString)
	if err != nil {
		c.lggr.Panic(err)
	}
	if invalid != "" {
		c.lggr.Error(invalid)
	}
	return v
}

// DefaultHTTPLimit defines the size limit for HTTP requests and responses
func (c *generalConfig) DefaultHTTPLimit() int64 {
	return c.viper.GetInt64(envvar.Name("DefaultHTTPLimit"))
}

// DefaultHTTPTimeout defines the default timeout for http requests
func (c *generalConfig) DefaultHTTPTimeout() models.Duration {
	return models.MustMakeDuration(getEnvWithFallback(c, envvar.NewDuration("DefaultHTTPTimeout")))
}
