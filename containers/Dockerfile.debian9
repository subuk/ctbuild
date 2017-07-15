FROM debian:9

ENV DEBIAN_FRONTEND noninteractive
ENV DEBIAN_PRIORITY critical
ENV DEBCONF_NOWARNINGS yes

RUN apt-get update && apt-get install -y wget dpkg-dev cdbs devscripts equivs fakeroot && apt-get clean
