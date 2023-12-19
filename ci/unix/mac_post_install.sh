#!/usr/bin/env bash

###############################################################################
# Note:
#
# This is an ERB template that is converted into a post-installation
# script run by `fpm`. It is to be used with the `fpm` arguments
# `--template-script` and `--template-value`.
#
# IT IS NOT MEANT TO BE RUN DIRECTLY!
#
# Expected variables:
#
#   - ps_path : string =>
#
#     The path to the Pennsieve installation, e.g. "/usr/local/opt/server",
#     "C:\Program Files\server", etc.
#
#   - ps_release_name : string =>
#
#     The name of the binary itself ("server")
#
#   - ps_version : string =>
#
#     The version string of the release ("0.1.x")
#
#   - ps_executable : string =>
#
#     The absolute path to the Pennsieve binary, e.g
#     /usr/local/opt/server/bin/${ps_release_name}
#
###############################################################################

PS_HOME="$HOME/.pennsieve"
PS_PATH="<%= ps_path %>"
PS_EXECUTABLE="<%= ps_executable %>"
USER_BINARY_DIR="/usr/local/bin"

# Create the Pennsieve home directory, if needed:
if [ ! -d "$PS_HOME" ]; then
	mkdir "$PS_HOME"
fi

INSTALL_LOG="$PS_HOME/install.log"

echo "Install log: $INSTALL_LOG"

echo "Installed $(date -u +"%Y-%m-%dT%H:%M:%SZ")" > $INSTALL_LOG
echo "PS_HOME=$PS_HOME" >> $INSTALL_LOG
echo "PS_PATH=<%= ps_path %>" >> $INSTALL_LOG
echo "PS_RELEASE_NAME=<%= ps_release_name %>" >> $INSTALL_LOG
echo "PS_VERSION=<%= ps_version %>" >> $INSTALL_LOG
echo "PS_EXECUTABLE=<%= ps_executable %>" >> $INSTALL_LOG

# Set the appropriate permissions:
if [ -n "$USER" ]; then
	sudo chown $USER:staff "$PS_HOME"
	sudo chown -R $USER:staff "$PS_HOME"
fi
sudo chmod 755 "$PS_PATH"
sudo chgrp staff "$PS_PATH"
sudo chgrp -R staff "$PS_PATH"

## Create the cache directory:
#if [ ! -d "$PS_HOME/cache" ]; then
#	mkdir "$PS_HOME/cache"
#fi

# Create /usr/local/bin if it does not exist
if [ ! -d $USER_BINARY_DIR ]; then

	sudo mkdir $USER_BINARY_DIR
	mkdir_status=$?
	if [ "$mkdir_status" -ne 0 ]; then
	  echo "Failed to create $USER_BINARY_DIR" >> $INSTALL_LOG
	fi

	sudo chmod 755 $USER_BINARY_DIR
  chmod_status=$?
  if [ "$chmod_status" -ne 0 ]; then
    echo "Failed to update permissions on $USER_BINARY_DIR" >> $INSTALL_LOG
  fi
fi


# Symlink $PS_EXECUTABLE to /usr/local/bin:
sudo ln -s -f "$PS_EXECUTABLE" "$USER_BINARY_DIR/pennsieve"
ln_status=$?
if [ "$ln_status" -ne 0 ]; then
  echo "Failed to create symlink from $PS_EXECUTABLE to $USER_BINARY_DIR/pennsieve" >> $INSTALL_LOG
fi

