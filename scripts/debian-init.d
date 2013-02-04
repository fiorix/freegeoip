#!/bin/sh

### BEGIN INIT INFO
# Provides:          freegeoip
# Required-Start:    $all
# Required-Stop:     $all
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: Starts a service on the cyclone web server
# Description:       Foobar
### END INIT INFO

PATH=/usr/local/sbin:/usr/local/bin:/sbin:/bin:/usr/sbin:/usr/bin
DAEMON=/usr/bin/twistd

SERVICE_DIR=/path/to/freegeoip
SERVICE_NAME=freegeoip

PYTHONPATH=$SERVICE_DIR:$PYTHONPATH
export PYTHONPATH

PORT=8888
LISTEN="127.0.0.1"
CONFIG=$SERVICE_DIR/$SERVICE_NAME.conf
PIDFILE=/var/run/$SERVICE_NAME.pid
LOGFILE=/var/log/$SERVICE_NAME.log
APP=${SERVICE_NAME}.web.Application

USER=www-data
GROUP=www-data
DAEMON_OPTS="-u $USER -g $GROUP --pidfile=$PIDFILE --logfile=$LOGFILE cyclone --port $PORT --listen $LISTEN --app $APP -c $CONFIG"

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
