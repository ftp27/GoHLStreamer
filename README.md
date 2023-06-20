# GoHLStreamer

GoHLStreamer is a Go-based server that streams video content in the HTTP Live Streaming (HLS) format. It allows you to convert MP4 files to HLS and serve them over HTTP. The server uses DigitalOcean Spaces for storing the MP4 files and the generated HLS files.

This project adds support for HTTP Live Streaming (HLS) to your Appwrite server. If you're not using Appwrite server, you can still use this project to load MP4 files from DigitalOcean Spaces (or another solution by defining the proper host). 

All files for streaming will be saved in the output directory on the Object Storage during the first load. Therefore, the initial load may take longer due to the export process. However, subsequent fetches will utilize the cached files from the Object Storage, resulting in faster loading times.

## Features

- Converts MP4 files to HLS format on-the-fly.
- Serves HLS streams for smooth video playback.
- Loading files from Appwrite server or DigitalOcean Spaces.
- Uses DigitalOcean Spaces for storing the media files.
- Easy to set up and configure.

## Prerequisites

- Go 1.15 or higher installed
- FFmpeg installed (used for MP4 to HLS conversion)
- DigitalOcean Spaces account with API credentials

## Installation

### Manual

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

If you want to use Appwrite instead of DigitalOcean Spaces, you can use the following environment variables:

```shell
APPWRITE_HOST=<Appwrite host>
APPWRITE_PROJECT=<Appwrite project ID>
APPWRITE_SECRET=<Appwrite secret key>
APPWRITE_BUCKET=<Appwrite storage bucket>
```

5. Run the server:

```shell
go run main.go
```

By default, the server will be available at `http://localhost:8080`. Adjust the `BASE_URL` value in the `.env` file to match your server's actual URL.

### Docker

To run the GoHLSStreamer server using Docker, follow these steps:

1. Install Docker on your machine by following the instructions specific to your operating system: [Docker Installation Guide](https://docs.docker.com/get-docker/).

2. Clone the repository:

   ```bash
   git clone https://github.com/ftp27/GoHLSStreamer.git
   ```

3. Navigate to the project directory:

   ```bash
   cd GoHLSStreamer
   ```

4. Build the Docker image:

   ```bash
   docker build -t gohlsstreamer .
   ```

   This command will build the Docker image based on the provided `Dockerfile`.

5. Run the Docker container:

   ```bash
   docker run  --env-file .env -p 8080:8080 gohlsstreamer
   ```

   This command will start the GoHLSStreamer server inside the Docker container and map port 8080 of the container to port 8080 on your local machine.

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