package api

import (
	"crypto/rand"
	"encoding/json"
	"math/big"

	"github.com/google/uuid"

	"github.com/status-im/status-go/account/generator"
	"github.com/status-im/status-go/eth-node/types"
	"github.com/status-im/status-go/multiaccounts/settings"
	"github.com/status-im/status-go/params"
	"github.com/status-im/status-go/protocol/identity/alias"
	"github.com/status-im/status-go/protocol/requests"
)

const pathWalletRoot = "m/44'/60'/0'/0"
const pathEIP1581 = "m/43'/60'/1581'"
const pathDefaultChat = pathEIP1581 + "/0'/0"
const pathDefaultWallet = pathWalletRoot + "/0"
const defaultMnemonicLength = 12
const walletAccountDefaultName = "Ethereum account"
const keystoreRelativePath = "keystore"

var paths = []string{pathWalletRoot, pathEIP1581, pathDefaultChat, pathDefaultWallet}

func defaultSettings(generatedAccountInfo generator.GeneratedAccountInfo, derivedAddresses map[string]generator.AccountInfo, mnemonic *string) (*settings.Settings, error) {
	chatKeyString := derivedAddresses[pathDefaultChat].PublicKey

	settings := &settings.Settings{}
	settings.Mnemonic = &generatedAccountInfo.Mnemonic
	settings.KeyUID = generatedAccountInfo.KeyUID
	settings.Address = types.HexToAddress(generatedAccountInfo.Address)
	settings.WalletRootAddress = types.HexToAddress(derivedAddresses[pathWalletRoot].Address)

	// Set chat key & name
	name, err := alias.GenerateFromPublicKeyString(chatKeyString)
	if err != nil {
		return nil, err
	}
	settings.Name = name
	settings.PublicKey = chatKeyString

	settings.DappsAddress = types.HexToAddress(derivedAddresses[pathDefaultWallet].Address)
	settings.EIP1581Address = types.HexToAddress(derivedAddresses[pathEIP1581].Address)
	settings.Mnemonic = mnemonic

	signingPhrase, err := buildSigningPhrase()
	if err != nil {
		return nil, err
	}
	settings.SigningPhrase = signingPhrase

	settings.SendPushNotifications = true
	settings.InstallationID = uuid.New().String()
	settings.UseMailservers = true

	settings.PreviewPrivacy = true
	settings.Currency = "usd"
	settings.ProfilePicturesVisibility = 1
	settings.LinkPreviewRequestEnabled = true

	visibleTokens := make(map[string][]string)
	visibleTokens["mainnet"] = []string{"SNT"}
	visibleTokensJSON, err := json.Marshal(visibleTokens)
	if err != nil {
		return nil, err
	}
	visibleTokenJSONRaw := json.RawMessage(visibleTokensJSON)
	settings.WalletVisibleTokens = &visibleTokenJSONRaw

	// TODO: fix this
	networks := make([]map[string]string, 0)
	networksJSON, err := json.Marshal(networks)
	if err != nil {
		return nil, err
	}
	networkRawMessage := json.RawMessage(networksJSON)
	settings.Networks = &networkRawMessage
	settings.CurrentNetwork = "mainnet_rpc"

	return settings, nil
}

func defaultNodeConfig(installationID string, request *requests.CreateAccount) (*params.NodeConfig, error) {
	// Set mainnet
	nodeConfig := &params.NodeConfig{}
	nodeConfig.NetworkID = 1
	nodeConfig.LogEnabled = request.LogEnabled
	nodeConfig.LogFile = request.LogFilePath
	nodeConfig.LogLevel = "ERROR"
	nodeConfig.DataDir = "/ethereum/mainnet_rpc"

	if request.LogLevel != nil {
		nodeConfig.LogLevel = *request.LogLevel
	}

	nodeConfig.APIModules = "wakuext,ext,waku"

	nodeConfig.UpstreamConfig = params.UpstreamRPCConfig{
		Enabled: true,
		URL:     "https://mainnet.infura.io/v3/800c641949d64d768a5070a1b0511938",
	}

	nodeConfig.Name = "StatusIM"
	nodeConfig.Rendezvous = false
	clusterConfig, err := params.LoadClusterConfigFromFleet("status.prod")
	if err != nil {
		return nil, err
	}
	nodeConfig.ClusterConfig = *clusterConfig
	nodeConfig.NoDiscovery = true
	nodeConfig.MaxPeers = 20
	nodeConfig.MaxPendingPeers = 20

	nodeConfig.WalletConfig = params.WalletConfig{Enabled: true}
	nodeConfig.LocalNotificationsConfig = params.LocalNotificationsConfig{Enabled: true}
	nodeConfig.BrowsersConfig = params.BrowsersConfig{Enabled: true}
	nodeConfig.PermissionsConfig = params.PermissionsConfig{Enabled: true}
	nodeConfig.MailserversConfig = params.MailserversConfig{Enabled: true}
	nodeConfig.EnableNTPSync = true

	nodes := []string{"enrtree://AOGECG2SPND25EEFMAJ5WF3KSGJNSGV356DSTL2YVLLZWIV6SAYBM@prod.nodes.status.im"}
	nodeConfig.ClusterConfig.WakuNodes = nodes
	nodeConfig.ClusterConfig.DiscV5BootstrapNodes = nodes

	nodeConfig.WakuV2Config = params.WakuV2Config{
		Enabled:        true,
		EnableDiscV5:   true,
		DiscoveryLimit: 20,
		Host:           "0.0.0.0",
		AutoUpdate:     true,
		PeerExchange:   true,
	}

	if request.WakuV2Nameserver != nil {
		nodeConfig.WakuV2Config.Nameserver = *request.WakuV2Nameserver
	}

	nodeConfig.ShhextConfig = params.ShhextConfig{
		InstallationID:             installationID,
		MaxMessageDeliveryAttempts: 6,
		MailServerConfirmations:    true,
		VerifyTransactionChainID:   1,
		DataSyncEnabled:            true,
		PFSEnabled:                 true,
	}

	if request.VerifyTransactionURL != nil {
		nodeConfig.ShhextConfig.VerifyTransactionURL = *request.VerifyTransactionURL
	}

	if request.VerifyENSURL != nil {
		nodeConfig.ShhextConfig.VerifyENSURL = *request.VerifyENSURL
	}

	if request.VerifyTransactionChainID != nil {
		nodeConfig.ShhextConfig.VerifyTransactionChainID = *request.VerifyTransactionChainID
	}

	if request.VerifyENSContractAddress != nil {
		nodeConfig.ShhextConfig.VerifyENSContractAddress = *request.VerifyENSContractAddress
	}

	if request.LogLevel != nil {
		nodeConfig.LogLevel = *request.LogLevel
		nodeConfig.LogEnabled = true

	} else {
		nodeConfig.LogEnabled = false
	}

	return nodeConfig, nil
}

func buildSigningPhrase() (string, error) {
	length := big.NewInt(int64(len(dictionary)))
	a, err := rand.Int(rand.Reader, length)
	if err != nil {
		return "", err
	}
	b, err := rand.Int(rand.Reader, length)
	if err != nil {
		return "", err
	}
	c, err := rand.Int(rand.Reader, length)
	if err != nil {
		return "", err
	}

	return dictionary[a.Int64()] + " " + dictionary[b.Int64()] + " " + dictionary[c.Int64()], nil

}

var dictionary = []string{
	"acid",
	"alto",
	"apse",
	"arch",
	"area",
	"army",
	"atom",
	"aunt",
	"babe",
	"baby",
	"back",
	"bail",
	"bait",
	"bake",
	"ball",
	"band",
	"bank",
	"barn",
	"base",
	"bass",
	"bath",
	"bead",
	"beak",
	"beam",
	"bean",
	"bear",
	"beat",
	"beef",
	"beer",
	"beet",
	"bell",
	"belt",
	"bend",
	"bike",
	"bill",
	"bird",
	"bite",
	"blow",
	"blue",
	"boar",
	"boat",
	"body",
	"bolt",
	"bomb",
	"bone",
	"book",
	"boot",
	"bore",
	"boss",
	"bowl",
	"brow",
	"bulb",
	"bull",
	"burn",
	"bush",
	"bust",
	"cafe",
	"cake",
	"calf",
	"call",
	"calm",
	"camp",
	"cane",
	"cape",
	"card",
	"care",
	"carp",
	"cart",
	"case",
	"cash",
	"cast",
	"cave",
	"cell",
	"cent",
	"chap",
	"chef",
	"chin",
	"chip",
	"chop",
	"chub",
	"chug",
	"city",
	"clam",
	"clef",
	"clip",
	"club",
	"clue",
	"coal",
	"coat",
	"code",
	"coil",
	"coin",
	"coke",
	"cold",
	"colt",
	"comb",
	"cone",
	"cook",
	"cope",
	"copy",
	"cord",
	"cork",
	"corn",
	"cost",
	"crab",
	"craw",
	"crew",
	"crib",
	"crop",
	"crow",
	"curl",
	"cyst",
	"dame",
	"dare",
	"dark",
	"dart",
	"dash",
	"data",
	"date",
	"dead",
	"deal",
	"dear",
	"debt",
	"deck",
	"deep",
	"deer",
	"desk",
	"dhow",
	"diet",
	"dill",
	"dime",
	"dirt",
	"dish",
	"disk",
	"dock",
	"doll",
	"door",
	"dory",
	"drag",
	"draw",
	"drop",
	"drug",
	"drum",
	"duck",
	"dump",
	"dust",
	"duty",
	"ease",
	"east",
	"eave",
	"eddy",
	"edge",
	"envy",
	"epee",
	"exam",
	"exit",
	"face",
	"fact",
	"fail",
	"fall",
	"fame",
	"fang",
	"farm",
	"fawn",
	"fear",
	"feed",
	"feel",
	"feet",
	"file",
	"fill",
	"film",
	"find",
	"fine",
	"fire",
	"fish",
	"flag",
	"flat",
	"flax",
	"flow",
	"foam",
	"fold",
	"font",
	"food",
	"foot",
	"fork",
	"form",
	"fort",
	"fowl",
	"frog",
	"fuel",
	"full",
	"gain",
	"gale",
	"galn",
	"game",
	"garb",
	"gate",
	"gear",
	"gene",
	"gift",
	"girl",
	"give",
	"glad",
	"glen",
	"glue",
	"glut",
	"goal",
	"goat",
	"gold",
	"golf",
	"gong",
	"good",
	"gown",
	"grab",
	"gram",
	"gray",
	"grey",
	"grip",
	"grit",
	"gyro",
	"hail",
	"hair",
	"half",
	"hall",
	"hand",
	"hang",
	"harm",
	"harp",
	"hate",
	"hawk",
	"head",
	"heat",
	"heel",
	"hell",
	"helo",
	"help",
	"hemp",
	"herb",
	"hide",
	"high",
	"hill",
	"hire",
	"hive",
	"hold",
	"hole",
	"home",
	"hood",
	"hoof",
	"hook",
	"hope",
	"hops",
	"horn",
	"hose",
	"host",
	"hour",
	"hunt",
	"hurt",
	"icon",
	"idea",
	"inch",
	"iris",
	"iron",
	"item",
	"jail",
	"jeep",
	"jeff",
	"joey",
	"join",
	"joke",
	"judo",
	"jump",
	"junk",
	"jury",
	"jute",
	"kale",
	"keep",
	"kick",
	"kill",
	"kilt",
	"kind",
	"king",
	"kiss",
	"kite",
	"knee",
	"knot",
	"lace",
	"lack",
	"lady",
	"lake",
	"lamb",
	"lamp",
	"land",
	"lark",
	"lava",
	"lawn",
	"lead",
	"leaf",
	"leek",
	"lier",
	"life",
	"lift",
	"lily",
	"limo",
	"line",
	"link",
	"lion",
	"lisa",
	"list",
	"load",
	"loaf",
	"loan",
	"lock",
	"loft",
	"long",
	"look",
	"loss",
	"lout",
	"love",
	"luck",
	"lung",
	"lute",
	"lynx",
	"lyre",
	"maid",
	"mail",
	"main",
	"make",
	"male",
	"mall",
	"manx",
	"many",
	"mare",
	"mark",
	"mask",
	"mass",
	"mate",
	"math",
	"meal",
	"meat",
	"meet",
	"menu",
	"mess",
	"mice",
	"midi",
	"mile",
	"milk",
	"mime",
	"mind",
	"mine",
	"mini",
	"mint",
	"miss",
	"mist",
	"moat",
	"mode",
	"mole",
	"mood",
	"moon",
	"most",
	"moth",
	"move",
	"mule",
	"mutt",
	"nail",
	"name",
	"neat",
	"neck",
	"need",
	"neon",
	"nest",
	"news",
	"node",
	"nose",
	"note",
	"oboe",
	"okra",
	"open",
	"oval",
	"oven",
	"oxen",
	"pace",
	"pack",
	"page",
	"pail",
	"pain",
	"pair",
	"palm",
	"pard",
	"park",
	"part",
	"pass",
	"past",
	"path",
	"peak",
	"pear",
	"peen",
	"peer",
	"pelt",
	"perp",
	"pest",
	"pick",
	"pier",
	"pike",
	"pile",
	"pimp",
	"pine",
	"ping",
	"pink",
	"pint",
	"pipe",
	"piss",
	"pith",
	"plan",
	"play",
	"plot",
	"plow",
	"poem",
	"poet",
	"pole",
	"polo",
	"pond",
	"pony",
	"poof",
	"pool",
	"port",
	"post",
	"prow",
	"pull",
	"puma",
	"pump",
	"pupa",
	"push",
	"quit",
	"race",
	"rack",
	"raft",
	"rage",
	"rail",
	"rain",
	"rake",
	"rank",
	"rate",
	"read",
	"rear",
	"reef",
	"rent",
	"rest",
	"rice",
	"rich",
	"ride",
	"ring",
	"rise",
	"risk",
	"road",
	"robe",
	"rock",
	"role",
	"roll",
	"roof",
	"room",
	"root",
	"rope",
	"rose",
	"ruin",
	"rule",
	"rush",
	"ruth",
	"sack",
	"safe",
	"sage",
	"sail",
	"sale",
	"salt",
	"sand",
	"sari",
	"sash",
	"save",
	"scow",
	"seal",
	"seat",
	"seed",
	"self",
	"sell",
	"shed",
	"shin",
	"ship",
	"shoe",
	"shop",
	"shot",
	"show",
	"sick",
	"side",
	"sign",
	"silk",
	"sill",
	"silo",
	"sing",
	"sink",
	"site",
	"size",
	"skin",
	"sled",
	"slip",
	"smog",
	"snob",
	"snow",
	"soap",
	"sock",
	"soda",
	"sofa",
	"soft",
	"soil",
	"song",
	"soot",
	"sort",
	"soup",
	"spot",
	"spur",
	"stag",
	"star",
	"stay",
	"stem",
	"step",
	"stew",
	"stop",
	"stud",
	"suck",
	"suit",
	"swan",
	"swim",
	"tail",
	"tale",
	"talk",
	"tank",
	"tard",
	"task",
	"taxi",
	"team",
	"tear",
	"teen",
	"tell",
	"temp",
	"tent",
	"term",
	"test",
	"text",
	"thaw",
	"tile",
	"till",
	"time",
	"tire",
	"toad",
	"toga",
	"togs",
	"tone",
	"tool",
	"toot",
	"tote",
	"tour",
	"town",
	"tram",
	"tray",
	"tree",
	"trim",
	"trip",
	"tuba",
	"tube",
	"tuna",
	"tune",
	"turn",
	"tutu",
	"twig",
	"type",
	"unit",
	"user",
	"vane",
	"vase",
	"vast",
	"veal",
	"veil",
	"vein",
	"vest",
	"vibe",
	"view",
	"vise",
	"wait",
	"wake",
	"walk",
	"wall",
	"wash",
	"wasp",
	"wave",
	"wear",
	"weed",
	"week",
	"well",
	"west",
	"whip",
	"wife",
	"will",
	"wind",
	"wine",
	"wing",
	"wire",
	"wish",
	"wolf",
	"wood",
	"wool",
	"word",
	"work",
	"worm",
	"wrap",
	"wren",
	"yard",
	"yarn",
	"yawl",
	"year",
	"yoga",
	"yoke",
	"yurt",
	"zinc",
	"zone",
}
