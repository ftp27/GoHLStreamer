# GoHLStreamer

GoHLStreamer is a Go-based server that streams video content in the HTTP Live Streaming (HLS) format. It allows you to convert MP4 files to HLS and serve them over HTTP. The server uses DigitalOcean Spaces for storing the MP4 files and the generated HLS files.

## Features

- Converts MP4 files to HLS format on-the-fly.
- Serves HLS streams for smooth video playback.
- Uses DigitalOcean Spaces for storing the media files.
- Easy to set up and configure.

## Prerequisites

- Go 1.15 or higher installed
- FFmpeg installed (used for MP4 to HLS conversion)
- DigitalOcean Spaces account with API credentials

## Installation

1. Clone the repository:

```shell
git clone https://github.com/ftp27/GoHLStreamer.git
```

2. Change into the project directory:

```shell
cd GoHLStreamer
```

3. Install the dependencies:

```shell
go mod download
```

4. Set up your environment by creating a `.env` file in the project directory with the following contents:

```shell
ENDPOINT=<DigitalOcean Spaces endpoint>
ACCESS_KEY_ID=<DigitalOcean Spaces access key>
SECRET_ACCESS_KEY=<DigitalOcean Spaces secret key>
BUCKET_NAME=<DigitalOcean Spaces bucket name>
BASE_URL=<Base URL where the server will be hosted>
TEMP_DIR=<Temporary directory path for HLS conversion>
INPUT_DIR=<Location of mp4 files in DigitalOcean Spaces>
OUTPUT_DIR=<Location of m3u8 files in DigitalOcean Spaces>
FFMPEG_PATH=<Path to the FFmpeg binary>
CACHE_SIZE=<The number of files can be kept keep local>
```

5. Run the server:

```shell
go run main.go
```

By default, the server will be available at `http://localhost:8080`. Adjust the `BASE_URL` value in the `.env` file to match your server's actual URL.

## Usage

1. Upload your MP4 files to the specified DigitalOcean Spaces bucket.

2. Access the HLS streams by appending `/hls/<object-name>/playlist.m3u8` to the server's URL.

Example:
```
http://your-server-url/hls/<object-name>/playlist.m3u8
```

## Contributing

Contributions are welcome! If you find any issues or have suggestions for improvements, please submit an issue or a pull request.

## License

This project is licensed under the [MIT License](LICENSE).