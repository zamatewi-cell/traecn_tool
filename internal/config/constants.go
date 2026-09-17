package config

// Trae CN backend API constants
const (
	AppID          = "6eefa01c-1036-4c7e-9ca5-d891f63bfcd8"
	IDEVersion     = "3.3.37"
	IDEVersionCode = "20260212"
	IDEVersionType = "stable"
	TrafficType    = "prod"
)

// Upstream base domains are vars so tests can swap in a local server.
var (
	AgentDomain = "https://trae-api-cn.mchost.guru"
	WSDomain    = "wss://trae-ws-cn.mchost.guru/custom_model"
)

// API endpoints
const (
	EndpointChatCompletion  = "/api/ide/v1/chat_completion"
	EndpointLLMRawChat      = "/api/ide/v1/llm_raw_chat"
	EndpointModelList       = "/api/ide/v1/model_list"
	EndpointGetDetailParam  = "/api/ide/v1/get_detail_param"
	EndpointAgentCreateTask = "/api/agent/v3/create_agent_task"
	EndpointAgentCommitTool = "/api/agent/v3/commit_toolcall_result"
	EndpointFeatures        = "/api/ide/v1/features"
	EndpointCodeCompletion  = "/api/ide/v1/code_completion_stream"
	EndpointEmbeddings      = "/api/ide/v1/embeddings"
	EndpointFastApply       = "/api/ide/v1/fast_apply"
	EndpointClientConfig    = "/api/ide/v1/get_client_config"
	EndpointPrivacy         = "/api/ide/v1/privacy/query"
	EndpointChatMode        = "/api/v1/commercial/chat_mode"
)

// Auth header names
const (
	HeaderIDEToken  = "X-IDE-Token"
	HeaderAuthToken = "X-Auth-Token"
	HeaderJWTToken  = "X-JWT-Token"
)

// Queue error codes
const (
	QueueExceedSize = 0xfd2 // 4050
	QueueTimeout    = 0xfd3 // 4051
)
