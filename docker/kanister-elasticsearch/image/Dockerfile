ARG TOOLS_IMAGE
FROM ${TOOLS_IMAGE} AS TOOLS_IMAGE

FROM debian:bullseye
COPY --from=TOOLS_IMAGE /usr/local/bin/restic /usr/local/bin/restic
COPY --from=TOOLS_IMAGE /usr/local/bin/kopia /usr/local/bin/kopia
ADD kando /usr/local/bin/

RUN apt update
RUN apt install -y npm bash curl libcap2-bin
RUN curl -fsSL https://deb.nodesource.com/setup_current.x | bash - && \
 apt-get install -y nodejs
RUN npm install -g npm yo grunt-cli bower express
RUN npm install elasticdump -g

RUN setcap cap_chown,cap_fowner,cap_dac_override+iep /usr/local/bin/kopia

CMD [ "/usr/bin/tail", "-f", "/dev/null" ]
