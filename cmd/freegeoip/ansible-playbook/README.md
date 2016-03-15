# Ansible Playbook for freegeoip.net

This is the ansible playbook used to deploy the public service
of freegeoip.net on Digital Ocean. It assumes your droplets
are Ubuntu 14.04 LTS and have private networking to each other.

It deploys the following:

- freegeoip web server
- redis server, for usage quota
- prometheus, for monitoring

Following is a TODO list for people willing to use this playbook:

- Replace the snakeoil certificate and key in `roles/freegeoip/files`
- Edit the `hosts` file and make sure each host has a private IP
- Edit `freegeoip.yml` and configure the variables accordingly

Then run:

```bash
ansible-playbook -u $user ./freegeoip.yml
```
