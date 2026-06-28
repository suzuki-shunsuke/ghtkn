FROM mirror.gcr.io/ubuntu:24.04@sha256:1e622c5f073b4f6bfad6632f2616c7f59ef256e96fe78bf6a595d1dc4376ac02
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y sudo ca-certificates curl vim
RUN echo 'foo ALL=(ALL) NOPASSWD: ALL' >> /etc/sudoers
RUN useradd -u 900 -m -r foo
USER foo
ENV PATH=/home/foo/.local/share/aquaproj-aqua/bin:$PATH
RUN mkdir /home/foo/workspace
WORKDIR /home/foo/workspace
RUN curl -sSfL -O https://raw.githubusercontent.com/aquaproj/aqua-installer/v4.0.5/aqua-installer
RUN echo "451028d56959cc738564885b1dbebc2691ea038ffde04e2472e4d486a3591146  aqua-installer" | sha256sum -c -
RUN chmod +x aqua-installer
RUN ./aqua-installer
RUN mkdir -p /home/foo/.config/ghtkn
