FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive

# Install everything the customer server needs: sshd, nginx, git, docker, curl, iproute2 (for ss)
RUN apt-get update && apt-get install -y \
    openssh-server \
    nginx \
    git \
    curl \
    iproute2 \
    docker.io \
    && rm -rf /var/lib/apt/lists/*

# Install Docker Compose v2 plugin — docker.io from apt doesn't bundle it.
# Detect arch so the image builds correctly on both amd64 and arm64.
RUN ARCH=$(uname -m) && \
    case "$ARCH" in \
        x86_64)  COMPOSE_ARCH="x86_64"  ;; \
        aarch64) COMPOSE_ARCH="aarch64" ;; \
    esac && \
    mkdir -p /usr/local/lib/docker/cli-plugins && \
    curl -fsSL "https://github.com/docker/compose/releases/download/v2.29.7/docker-compose-linux-${COMPOSE_ARCH}" \
        -o /usr/local/lib/docker/cli-plugins/docker-compose && \
    chmod +x /usr/local/lib/docker/cli-plugins/docker-compose

# Create the ubuntu user and add it to the docker group so SSH sessions can run docker commands
RUN useradd -m -s /bin/bash ubuntu \
    && usermod -aG docker ubuntu \
    && mkdir -p /home/ubuntu/.ssh \
    && chmod 700 /home/ubuntu/.ssh

# sshd needs this directory and host keys (ssh-keygen -A in entrypoint handles the keys)
RUN mkdir -p /var/run/sshd

# Allow pubkey auth; disable root login
RUN sed -i 's/#PubkeyAuthentication yes/PubkeyAuthentication yes/' /etc/ssh/sshd_config \
    && sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin no/' /etc/ssh/sshd_config

# Replace nginx.conf so our include directive is inside the http block.
# Give ubuntu ownership so the Launchpad SSH session can write service configs
# and modify nginx.conf without sudo (test environment only).
COPY customer/nginx.conf /etc/nginx/nginx.conf
RUN mkdir -p /etc/nginx/launchpad \
    && chown ubuntu:ubuntu /etc/nginx/nginx.conf \
    && chown -R ubuntu:ubuntu /etc/nginx/launchpad \
    && chown -R ubuntu:ubuntu /var/log/nginx \
    && chown -R ubuntu:ubuntu /var/lib/nginx \
    && touch /run/nginx.pid && chown ubuntu:ubuntu /run/nginx.pid

# Copy the test app — the agent will git clone this and docker build it
COPY testapp /home/ubuntu/testapp
RUN chown -R ubuntu:ubuntu /home/ubuntu/testapp

COPY customer/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 22 80

CMD ["/entrypoint.sh"]
