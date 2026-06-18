package types

import (
	"encoding/json"
	"strconv"
	"time"
)

const DiscordEpoch int64 = 1420070400000

type Snowflake uint64

func (s Snowflake) String() string {
	return strconv.FormatUint(uint64(s), 10)
}

func (s Snowflake) Int64() int64 {
	return int64(s)
}

func (s Snowflake) Timestamp() time.Time {
	ms := (int64(s) >> 22) + DiscordEpoch
	return time.UnixMilli(ms)
}

func (s Snowflake) WorkerID() int {
	return int((s >> 17) & 0x1F)
}

func (s Snowflake) ProcessID() int {
	return int((s >> 12) & 0x1F)
}

func (s Snowflake) Increment() int {
	return int(s & 0xFFF)
}

func (s Snowflake) IsZero() bool {
	return s == 0
}

func ParseSnowflake(s string) (Snowflake, error) {
	if s == "" {
		return 0, nil
	}
	i, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return Snowflake(i), nil
}

func MustParseSnowflake(s string) Snowflake {
	sf, err := ParseSnowflake(s)
	if err != nil {
		panic(err)
	}
	return sf
}

func (s Snowflake) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *Snowflake) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		sf, err := ParseSnowflake(str)
		if err != nil {
			return err
		}
		*s = sf
		return nil
	}

	var num uint64
	if err := json.Unmarshal(data, &num); err != nil {
		return err
	}
	*s = Snowflake(num)
	return nil
}

func NewSnowflake(id uint64) Snowflake {
	return Snowflake(id)
}

func GenerateNonce() string {
	ms := time.Now().UnixMilli()
	nonce := (ms - DiscordEpoch) << 22
	return strconv.FormatInt(nonce, 10)
}
