PREFIX = /usr/local/bin

all:
	@echo Run \'make install\' to install distrobox. Run \'make uninstall\' to uninstall.

install:
	@./install -p ${PREFIX}

uninstall:
	@./uninstall -p ${PREFIX}

.PHONY: all install uninstall
