package turns

// TurnDataKey is a typed string key for Turn.Data map.
type TurnDataKey string

// TurnMetadataKey is a typed string key for Turn.Metadata map.
type TurnMetadataKey string

// BlockMetadataKey is a typed string key for Block.Metadata map.
type BlockMetadataKey string

// RunMetadataKey is a typed string key for Run.Metadata map.
type RunMetadataKey string

// String returns the underlying string value for logging and serialization.
func (k TurnDataKey) String() string {
	return string(k)
}

// String returns the underlying string value for logging and serialization.
func (k TurnMetadataKey) String() string {
	return string(k)
}

// String returns the underlying string value for logging and serialization.
func (k BlockMetadataKey) String() string {
	return string(k)
}

// String returns the underlying string value for logging and serialization.
func (k RunMetadataKey) String() string {
	return string(k)
}


