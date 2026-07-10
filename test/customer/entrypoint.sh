#!/bin/bash
set -e

# Load the SSH public key that drive.sh generated — mounted at runtime so the
# image itself never bakes in a key.
mkdir -p /home/ubuntu/.ssh
if [ -f /tmp/launchpad_key.pub ]; then
    cp /tmp/launchpad_key.pub /home/ubuntu/.ssh/authorized_keys
    chmod 600 /home/ubuntu/.ssh/authorized_keys
    chown -R ubuntu:ubuntu /home/ubuntu/.ssh
fi

# Generate SSH host keys if this is a fresh container.
ssh-keygen -A

# Start Docker daemon in the background (DinD — container must run privileged).
# vfs storage driver works in nested Docker environments where overlayfs is unavailable.
dockerd --storage-driver vfs 2>/var/log/dockerd.log &

echo "Waiting for Docker daemon..."
until docker info >/dev/null 2>&1; do sleep 1; done
echo "Docker ready"

# Initialise the test app as a git repo so the agent can clone it via file://.
# Run as ubuntu (who owns the directory) to avoid git's dubious-ownership check.
if [ ! -d /home/ubuntu/testapp/.git ]; then
    su ubuntu -c "git -C /home/ubuntu/testapp -c user.email=test@launchpad.dev -c user.name=Test init -b main"
    su ubuntu -c "git -C /home/ubuntu/testapp -c user.email=test@launchpad.dev -c user.name=Test add ."
    su ubuntu -c "git -C /home/ubuntu/testapp -c user.email=test@launchpad.dev -c user.name=Test commit -m 'Initial commit'"
fi

# Start nginx as ubuntu so the SSH session (also ubuntu) can send nginx -s reload.
su ubuntu -c 'nginx'

# Start sshd in the foreground — keeps the container alive.
exec /usr/sbin/sshd -D
