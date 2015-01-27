# Ansible Playbook

This is an ansible playbook for freegeoip and redis.

If you plan to use SSL, replace the snakeoil certificate and key
(the `.pem` files) in `roles/webserver/files/` with real ones.

Following is a list of other files to be edited prior to using this:

- hosts: to add your servers
- roles/webserver/vars/main.yml: to configure the freegeoip daemon
- roles/webserver/files/nginx.conf: to increase # of workers, etc
- roles/redis/files/iptables.conf: to add your web servers (optional)

Then run:

	ansible-playbook -u root ./freegeoip.yml
