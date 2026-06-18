package props

type Capabilities uint64

const (
	CapabilityLazyUserNotes                      Capabilities = 1 << 0
	CapabilityNoAffineUserIDs                    Capabilities = 1 << 1
	CapabilityVersionedReadStates                Capabilities = 1 << 2
	CapabilityVersionedUserGuildSettings         Capabilities = 1 << 3
	CapabilityDedupeUserObjects                  Capabilities = 1 << 4
	CapabilityPrioritizedReadyPayload            Capabilities = 1 << 5
	CapabilityMultipleGuildExperimentPopulations Capabilities = 1 << 6
	CapabilityNonChannelReadStates               Capabilities = 1 << 7
	CapabilityAuthTokenRefresh                   Capabilities = 1 << 8
	CapabilityUserSettingsProto                  Capabilities = 1 << 9
	CapabilityClientStateV2                      Capabilities = 1 << 10
	CapabilityPassiveGuildUpdate                 Capabilities = 1 << 11
	CapabilityUnknown12                          Capabilities = 1 << 12
	CapabilityUnknown13                          Capabilities = 1 << 13
	CapabilityUnknown14                          Capabilities = 1 << 14
)

func DefaultCapabilities() Capabilities {
	return CapabilityLazyUserNotes |
		CapabilityNoAffineUserIDs |
		CapabilityVersionedReadStates |
		CapabilityVersionedUserGuildSettings |
		CapabilityDedupeUserObjects |
		CapabilityPrioritizedReadyPayload |
		CapabilityMultipleGuildExperimentPopulations |
		CapabilityNonChannelReadStates |
		CapabilityAuthTokenRefresh |
		CapabilityUserSettingsProto |
		CapabilityClientStateV2 |
		CapabilityPassiveGuildUpdate
}

func (c Capabilities) Value() int {
	return int(c)
}

func (c Capabilities) Has(cap Capabilities) bool {
	return (c & cap) == cap
}

func (c Capabilities) Add(cap Capabilities) Capabilities {
	return c | cap
}

func (c Capabilities) Remove(cap Capabilities) Capabilities {
	return c &^ cap
}

func (c Capabilities) String() string {
	return string(rune(c.Value()))
}
