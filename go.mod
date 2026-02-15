module github.com/alesr/tidskott-pi

go 1.25.6

replace github.com/alesr/tidskott-core => ../tidskott-core

replace github.com/alesr/tidskott-camera-pi => ../tidskott-camera-pi

replace github.com/alesr/tidskott-uploader => ../tidskott-uploader

require (
	github.com/alesr/tidskott-core v0.0.0-00010101000000-000000000000
	github.com/alesr/tidskott-uploader v0.0.0-00010101000000-000000000000
	github.com/pelletier/go-toml/v2 v2.2.4
)

require github.com/oklog/ulid/v2 v2.1.1 // indirect

replace github.com/alesr/tidskott-pi => .
