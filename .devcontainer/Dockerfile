FROM mcr.microsoft.com/devcontainers/base:ubuntu-22.04

ARG NODE_VERSION=20
ARG GO_VERSION=1.23.4

# Node.js
RUN curl -fsSL https://deb.nodesource.com/setup_${NODE_VERSION}.x | bash - \
    && apt-get install -y nodejs \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# Go
RUN curl -fsSL https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz \
    | tar -C /usr/local -xz
ENV PATH=$PATH:/usr/local/go/bin

# AWS CLI v2
RUN curl -fsSL "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o /tmp/awscli.zip \
    && unzip /tmp/awscli.zip -d /tmp \
    && /tmp/aws/install \
    && rm -rf /tmp/aws /tmp/awscli.zip

# AWS SAM CLI
RUN curl -fsSL "https://github.com/aws/aws-sam-cli/releases/latest/download/aws-sam-cli-linux-x86_64.zip" -o /tmp/samcli.zip \
    && unzip /tmp/samcli.zip -d /tmp/samcli \
    && /tmp/samcli/install \
    && rm -rf /tmp/samcli /tmp/samcli.zip

# Claude Code CLI
RUN npm install -g @anthropic-ai/claude-code

# Angular CLI
RUN npm install -g @angular/cli
