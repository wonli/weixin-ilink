package ilinkapi

type CDNMedia struct {
	EncryptQueryParam string `json:"encrypt_query_param,omitempty"`
	AESKey            string `json:"aes_key,omitempty"`
	EncryptType       int    `json:"encrypt_type,omitempty"`
}

type ImageItem struct {
	Media       *CDNMedia `json:"media,omitempty"`
	ThumbMedia  *CDNMedia `json:"thumb_media,omitempty"`
	AESKeyHex   string    `json:"aeskey,omitempty"`
	URL         string    `json:"url,omitempty"`
	MidSize     int64     `json:"mid_size,omitempty"`
	ThumbSize   int64     `json:"thumb_size,omitempty"`
	ThumbHeight int       `json:"thumb_height,omitempty"`
	ThumbWidth  int       `json:"thumb_width,omitempty"`
	HDSize      int64     `json:"hd_size,omitempty"`
}

type VoiceItem struct {
	Media         *CDNMedia `json:"media,omitempty"`
	EncodeType    int       `json:"encode_type,omitempty"`
	BitsPerSample int       `json:"bits_per_sample,omitempty"`
	SampleRate    int       `json:"sample_rate,omitempty"`
	Playtime      int       `json:"playtime,omitempty"`
	Text          string    `json:"text,omitempty"`
}

type FileItem struct {
	Media    *CDNMedia `json:"media,omitempty"`
	FileName string    `json:"file_name,omitempty"`
	MD5      string    `json:"md5,omitempty"`
	Len      string    `json:"len,omitempty"`
}

type VideoItem struct {
	Media       *CDNMedia `json:"media,omitempty"`
	VideoSize   int64     `json:"video_size,omitempty"`
	PlayLength  int       `json:"play_length,omitempty"`
	VideoMD5    string    `json:"video_md5,omitempty"`
	ThumbMedia  *CDNMedia `json:"thumb_media,omitempty"`
	ThumbSize   int64     `json:"thumb_size,omitempty"`
	ThumbHeight int       `json:"thumb_height,omitempty"`
	ThumbWidth  int       `json:"thumb_width,omitempty"`
}

type UploadedMedia struct {
	FileKey                     string
	DownloadEncryptedParam      string
	ThumbDownloadEncryptedParam string
	AESKeyHex                   string
	MD5Hex                      string
	FileSize                    int
	FileSizeCiphertextBytes     int
}
