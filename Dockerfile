# Stage 1: Builder
# 根本的な問題を解決するため、依存ライブラリはすべてソースからビルドするアプローチに変更
FROM golang:1.24.5-bookworm AS builder

WORKDIR /build

# --- 1. 依存関係のインストール ---
# posixスレッド対応版のコンパイラを明示的にインストールする
RUN apt-get update && apt-get install -y --no-install-recommends \
    wget ca-certificates \
    build-essential pkg-config \
    autotools-dev libtool automake cmake \
    nasm \
    g++-mingw-w64-x86-64-posix \
    gcc-mingw-w64-x86-64-posix \
    && rm -rf /var/lib/apt/lists/*

# --- 2. クロスコンパイル環境の定義 ---
ENV GOOS=windows
ENV GOARCH=amd64
ENV CGO_ENABLED=1
ENV CC=x86_64-w64-mingw32-gcc-posix
ENV CXX=x86_64-w64-mingw32-g++-posix
ENV MINGW_PREFIX=/usr/x86_64-w64-mingw32
ENV PKG_CONFIG_PATH=${MINGW_PREFIX}/lib/pkgconfig
ENV CGO_CFLAGS="-I${MINGW_PREFIX}/include"
ENV CGO_LDFLAGS="-L${MINGW_PREFIX}/lib"

# --- 3. 依存ライブラリのソースからのビルド (依存順) ---

# 3.1 zlib
RUN ZLIB_VERSION=1.3.1 && \
    wget https://www.zlib.net/zlib-${ZLIB_VERSION}.tar.gz && \
    tar -zxvf zlib-${ZLIB_VERSION}.tar.gz && \
    cd zlib-${ZLIB_VERSION} && \
    CHOST=x86_64-w64-mingw32 ./configure --prefix=${MINGW_PREFIX} --static && \
    make -j$(nproc) && \
    make install

# 3.2 libjpeg-turbo
RUN LIBJPEG_TURBO_VERSION=3.0.3 && \
    wget https://github.com/libjpeg-turbo/libjpeg-turbo/archive/refs/tags/${LIBJPEG_TURBO_VERSION}.tar.gz && \
    tar -zxvf ${LIBJPEG_TURBO_VERSION}.tar.gz && \
    cd libjpeg-turbo-${LIBJPEG_TURBO_VERSION} && \
    cmake . \
        -DCMAKE_SYSTEM_NAME=Windows \
        -DCMAKE_SYSTEM_PROCESSOR=x86_64 \
        -DCMAKE_C_COMPILER=${CC} \
        -DCMAKE_CXX_COMPILER=${CXX} \
        -DCMAKE_RC_COMPILER=x86_64-w64-mingw32-windres \
        -DCMAKE_INSTALL_PREFIX=${MINGW_PREFIX} \
        -DENABLE_STATIC=ON -DENABLE_SHARED=OFF -DWITH_JPEG8=1 && \
    make -j$(nproc) && \
    make install

# 3.3 libpng (zlibに依存)
RUN LIBPNG_VERSION=1.6.43 && \
    wget https://download.sourceforge.net/libpng/libpng-${LIBPNG_VERSION}.tar.gz && \
    tar -zxvf libpng-${LIBPNG_VERSION}.tar.gz && \
    cd libpng-${LIBPNG_VERSION} && \
    ./configure --host=x86_64-w64-mingw32 --prefix=${MINGW_PREFIX} --enable-static --disable-shared && \
    make -j$(nproc) && \
    make install

# 3.4 libtiff (zlib, jpegに依存)
RUN LIBTIFF_VERSION=4.6.0 && \
    wget https://download.osgeo.org/libtiff/tiff-${LIBTIFF_VERSION}.tar.gz && \
    tar -zxvf tiff-${LIBTIFF_VERSION}.tar.gz && \
    cd tiff-${LIBTIFF_VERSION} && \
    ./configure --host=x86_64-w64-mingw32 --prefix=${MINGW_PREFIX} --enable-static --disable-shared --disable-lzma --disable-jbig && \
    make -j$(nproc) && \
    make install

# 3.5 libwebp (png, jpeg, tiffに依存)
RUN LIBWEBP_VERSION=1.4.0 && \
    wget https://storage.googleapis.com/downloads.webmproject.org/releases/webp/libwebp-${LIBWEBP_VERSION}.tar.gz && \
    tar -zxvf libwebp-${LIBWEBP_VERSION}.tar.gz && \
    cd libwebp-${LIBWEBP_VERSION} && \
    ./configure --host=x86_64-w64-mingw32 --prefix=${MINGW_PREFIX} --enable-static --disable-shared --enable-everything && \
    make -j$(nproc) && \
    make install

# 3.6 Leptonica (上記のすべてに依存)
# 【修正】Leptonicaのビルド時に、依存ライブラリを明示的に指定する
RUN LEPTONICA_VERSION=1.84.1 && \
    wget --no-check-certificate http://www.leptonica.org/source/leptonica-${LEPTONICA_VERSION}.tar.gz && \
    tar -zxvf leptonica-${LEPTONICA_VERSION}.tar.gz && \
    cd leptonica-${LEPTONICA_VERSION} && \
    LIBS="-lpng -ljpeg -ltiff -lwebp -lz" ./configure \
        --host=x86_64-w64-mingw32 \
        --prefix=${MINGW_PREFIX} \
        --enable-static \
        --disable-shared && \
    make -j$(nproc) && \
    make install

# 3.7 Tesseract (Leptonicaに依存)
# 【修正】Tesseractのビルド時に、必要なシステムライブラリを明示的に指定する
RUN TESSERACT_VERSION=5.3.4 && \
    wget https://github.com/tesseract-ocr/tesseract/archive/refs/tags/${TESSERACT_VERSION}.tar.gz && \
    tar -zxvf ${TESSERACT_VERSION}.tar.gz && \
    cd tesseract-${TESSERACT_VERSION} && \
    ./autogen.sh && \
    LIBS="-lgomp -lws2_32" ./configure \
        --host=x86_64-w64-mingw32 \
        --prefix=${MINGW_PREFIX} \
        --enable-static \
        --disable-shared \
        LIBLEPT_LIBS="-L${MINGW_PREFIX}/lib -llept" \
        CPPFLAGS="-I${MINGW_PREFIX}/include" && \
    make -j$(nproc) && \
    make install

# --- 4. Tesseract言語データのダウンロード ---
RUN mkdir -p /tessdata && \
    wget -P /tessdata https://github.com/tesseract-ocr/tessdata_fast/raw/main/eng.traineddata && \
    wget -P /tessdata https://github.com/tesseract-ocr/tessdata_fast/raw/main/jpn.traineddata

# --- 5. Goアプリケーションのビルド ---
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# このGoビルドコマンドは修正不要。上記までのCライブラリのビルドが正しければ、pkg-configが依存関係を解決してくれる。
RUN go build -v -tags gosseract_static -ldflags="-s -w" -o nightreign-relics-counter.exe ./main.go

# --- Final Stage ---
FROM scratch
WORKDIR /app
COPY --from=builder /app/nightreign-relics-counter.exe .
COPY --from=builder /tessdata /app/tessdata
# Windows用の正しいDLLをコピーする
COPY --from=builder /usr/x86_64-w64-mingw32/bin/libwinpthread-1.dll .
COPY --from=builder /usr/x86_64-w64-mingw32/bin/libstdc++-6.dll .
COPY --from=builder /usr/x86_64-w64-mingw32/bin/libgcc_s_seh-1.dll .
