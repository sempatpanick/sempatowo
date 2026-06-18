package voice

// Voice Opcodes
const (
	// OpIdentify is sent to identify to the voice gateway.
	OpIdentify = 0
	// OpSelectProtocol is sent to select the protocol (UDP).
	OpSelectProtocol = 1
	// OpReady is received when the client is ready, containing heartbeat info and IP/Port.
	OpReady = 2
	// OpHeartbeat is sent to keep the connection alive.
	OpHeartbeat = 3
	// OpSessionDescription is received with encryption keys and mode.
	OpSessionDescription = 4
	// OpSpeaking is sent or received to indicate speaking status.
	OpSpeaking = 5
	// OpHeartbeatACK is received as a heartbeat acknowledgment.
	OpHeartbeatACK = 6
	// OpResume is sent to resume the session.
	OpResume = 7
	// OpHello is received with the heartbeat interval.
	OpHello = 8
	// OpResumed is received when the session is effectively resumed.
	OpResumed = 9
	// OpClientDisconnect is received when a user disconnects from voice.
	OpClientDisconnect = 13
)

// Voice Close Codes
const (
	CloseUnknownOpcode         = 4001
	CloseNotAuthenticated      = 4003
	CloseAuthenticationFailed  = 4004
	CloseAlreadyAuthenticated  = 4005
	CloseSessionNoLongerValid  = 4006
	CloseSessionTimeout        = 4009
	CloseServerNotFound        = 4011
	CloseUnknownProtocol       = 4012
	CloseDisconnected          = 4014
	CloseVoiceServerCrash      = 4015
	CloseUnknownEncryptionMode = 4016
)
