FROM python@sha256:f77ac9e44ae96ef2c90b8053ea08c31f8be030f824196b0ae4db6d462c84e51f AS build
COPY requirements.lock model-manifest.json download_model.py /build/
RUN python -m pip install --no-cache-dir --require-hashes --only-binary=:all: --target /deps -r /build/requirements.lock
ADD --checksum=sha256:6255977466101d92ec2886b967aa4d27f67161606fa6552fbb11d9772c650cce https://codeload.github.com/keeltrace/hermes-nerve/tar.gz/b9e78dd5e00cf1117c563cada4d563a56f66e609 /build/nerve.tar.gz
RUN mkdir /nerve && tar -xzf /build/nerve.tar.gz --strip-components=1 -C /nerve && rm /build/nerve.tar.gz
RUN python /build/download_model.py

FROM python@sha256:f77ac9e44ae96ef2c90b8053ea08c31f8be030f824196b0ae4db6d462c84e51f
COPY --from=build /deps /deps
COPY --from=build /nerve /nerve
COPY --from=build /model /model
RUN mkdir -p /opt/laya && chmod 0755 /opt/laya
COPY --chmod=0444 model-manifest.json /opt/laya/model-manifest.json
LABEL io.repokit.laya.recipe="1" \
      io.repokit.nerve.revision="b9e78dd5e00cf1117c563cada4d563a56f66e609" \
      io.repokit.laya.model-revision="1a793eb568e6718f15941d08f85432581df534e3"
ENV PYTHONPATH=/deps:/nerve HOME=/tmp USER=laya USE_TF=0 \
    HF_HUB_OFFLINE=1 TRANSFORMERS_OFFLINE=1 PYTHONDONTWRITEBYTECODE=1 \
    TORCHINDUCTOR_CACHE_DIR=/tmp/torchinductor OMP_NUM_THREADS=4 MKL_NUM_THREADS=4
WORKDIR /nerve
USER 1000:1000
HEALTHCHECK --interval=5s --timeout=3s --start-period=10s --retries=24 CMD python -c "import json,urllib.request; assert json.load(urllib.request.urlopen('http://127.0.0.1:8765/healthz',timeout=2)) == {'ok':True,'provider':'Laya','model':'/model','transport':'laya-local-http'}"
CMD ["python", "-m", "hermes_nerve.reflex.laya_service", "--host", "127.0.0.1", "--port", "8765", "--device", "cpu", "--model", "/model"]
