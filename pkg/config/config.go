package config

var (
	// ImageConfigName is the name for image configuration file (for Packer) that
	// will be generated by enzyme with redefinition variables in source image template
	ImageConfigName string
	// ClusterConfigName is the name for cluster configuration file (for Terraform) that
	// will be generated by enzyme with redefinition variables in source cluster template
	ClusterConfigName string
)

func init() {
	ImageConfigName = "config.json"
	ClusterConfigName = "config.tf.json"
}

// Config is base interface for the config package
type Config interface {
	Keys() []string
	IsSet(key string) bool

	GetString(key string) (string, error)
	GetStringMap(key string) (map[string]interface{}, error)

	GetValue(key string) (interface{}, error)
	SetValue(key string, value interface{})

	Serialize(configName string) error
	Deserialize(configName string) error
	DeserializeBytes(jsonContent []byte) error

	DeleteKey(key string) error

	Copy() (Config, error)
	Print()
}
