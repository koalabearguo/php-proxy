
include $(TOPDIR)/rules.mk

PKG_NAME:=php-proxy
PKG_VERSION:=2.1.4
PKG_RELEASE:=$(AUTORELEASE)

PKG_SOURCE:=$(PKG_NAME)-$(PKG_VERSION).tar.gz
PKG_SOURCE_URL:=https://codeload.github.com/koalabearguo/php-proxy/tar.gz/v$(PKG_VERSION)?
PKG_HASH:=203d74a8c4b2dec28ab07e247f14737dd1edf110a0c6f3805d5e3634fc7545f6

PKG_LICENSE:=MIT
PKG_LICENSE_FILE:=LICENSE
PKG_MAINTAINER:=koala <koalabearguo@gmail.com>

PKG_BUILD_DEPENDS:=golang/host
PKG_BUILD_PARALLEL:=1
PKG_USE_MIPS16:=0

GO_PKG:=github.com/koalabearguo/php-proxy

include $(INCLUDE_DIR)/package.mk
include $(TOPDIR)/feeds/packages/lang/golang/golang-package.mk

define Package/php-proxy/Default
  SECTION:=net
  CATEGORY:=Network
  #SUBMENU:=File Transfer
  TITLE:=php-proxy for agent web content via php/js
  URL:=https://github.com/koalabearguo/php-proxy
endef

define Package/php-proxy
  $(call Package/php-proxy/Default)
  DEPENDS:=$(GO_ARCH_DEPENDS)
endef

define Package/php-proxy/description
	php-proxy for agent web content via php/js
endef

define Package/php-proxy/conffiles
/etc/php-proxy/php-proxy.json
endef

define Build/Prepare
	$(call Build/Prepare/Default)
	(cd $(PKG_BUILD_DIR); \
	 $(RM) -fr php-proxy.syso; \
	 echo "module github.com/koalabearguo/php-proxy" > go.mod; \
	 echo "" >> go.mod; \
	 echo "go 1.15" >> go.mod \
	)
endef

define Package/php-proxy/install
	$(call GoPackage/Package/Install/Bin,$(PKG_INSTALL_DIR))
	$(INSTALL_DIR) $(1)/usr/bin/
	$(INSTALL_BIN) $(PKG_INSTALL_DIR)/usr/bin/php-proxy $(1)/usr/bin/
	$(INSTALL_DIR) $(1)/etc/php-proxy
	cat $(PKG_BUILD_DIR)/php-proxy.json > $(1)/etc/php-proxy/php-proxy.json
endef

$(eval $(call GoBinPackage,php-proxy))
$(eval $(call BuildPackage,php-proxy))
