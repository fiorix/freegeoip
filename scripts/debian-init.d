#!/bin/sh

### BEGIN INIT INFO
# Provides:          foobar
# Required-Start:    $all
# Required-Stop:     $all
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: Starts a service for the Twisted plugin 'foobar'
# Description:       Foobar
### END INIT INFO

PATH=/usr/local/sbin:/usr/local/bin:/sbin:/bin:/usr/sbin:/usr/bin
DAEMON=/usr/bin/twistd

SERVICE_DIR=/path/to/foobar
SERVICE_NAME=foobar

PORT=8888
LISTEN="127.0.0.1"
CONFIG=/path/to/$SERVICE_NAME.conf
PIDFILE=/var/run/$SERVICE_NAME.pid
LOGFILE=/var/log/$SERVICE_NAME.log

DAEMON_OPTS="--pidfile=$PIDFILE --logfile=$LOGFILE $SERVICE_NAME -p $PORT -l $LISTEN -c $CONFIG"

# Set python path so twistd can find the plugin
# See: http://twistedmatrix.com/projects/core/documentation/howto/plugin.html
export PYTHONPATH=$SERVICE_DIR

if [ ! -x $DAEMON ]; then
  echo "ERROR: Can't execute $DAEMON."
  exit 1
fi

if [ ! -d $SERVICE_DIR ]; then
  echo "ERROR: Directory doesn't exist: $SERVICE_DIR"
  exit 1
fi

start_service() {
  echo -n " * Starting $SERVICE_NAME... "
  start-stop-daemon -Sq -p $PIDFILE -x $DAEMON -- $DAEMON_OPTS
  e=$?
  if [ $e -eq 1 ]; then
    echo "already running"
    return
  fi

  if [ $e -eq 255 ]; then
    echo "couldn't start"
    return
  fi

  echo "done"
}

stop_service() {
  echo -n " * Stopping $SERVICE_NAME... "
  start-stop-daemon -Kq -R 10 -p $PIDFILE
  e=$?
  if [ $e -eq 1 ]; then
    echo "not running"
    return
  fi

  echo "done"
}

case "$1" in
  start)
    start_service
    ;;
  stop)
    stop_service
    ;;
  restart)
    stop_service
    start_service
    ;;
  *)
    echo "Usage: /etc/init.d/$SERVICE_NAME {start|stop|restart}" >&2
    exit 1
    ;;
esac

exit 0
