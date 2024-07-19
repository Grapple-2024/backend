SSL certificates are currently managed and rotated manually.

  The current certificate was generating on July 19th and is valid until Thursday, October 17, 2024 at 12:03:09 PM.


# Certificate rotation guide

Follow the guide below to generate a new certificate for Grapple.

1. Download `certbot` and the `certbot-dns-route53` plugin: https://certbot-dns-route53.readthedocs.io/en/stable/
2. Generate/rotate existing certificate:
```sh
certbot certonly \
-n \
--agree-tos \
--email jordan@dionysustechnologygroup.com \
-d grapplemma.com \
-d '*.grapplemma.com' \
--dns-route53 \
--preferred-challenges=dns \
--logs-dir /tmp/letsencrypt \
--config-dir ~/local/letsencrypt \
--work-dir /tmp/letsencrypt --key-type rsa --cert-name grapplemma.com
```
3. The certbot CLI will generate the certificates on your local machine. The next step is to copy them over to the EC2 instance that is hosting the Grapple frontend.
4. `scp ~/local/letsencrypt/live/grapplemma.com/fullchain.pem grapple:/home/ec2-user/fullchain.pem`
4. `scp ~/local/letsencrypt/live/grapplemma.com/privkey.pem grapple:/home/ec2-user/privkey.pem`
5. Next, login to the EC2 instance and run `sudo su`:
6. `ssh grapple` -> `sudo su`
7. Next, copy the files from /home/ec2-user/ to the letsencrypt directory:
```sh
cp /home/ec2-user/privkey.pem /root/local/letsencrypt/live/grapplemma.com/privkey.pem
cp /home/ec2-user/fullchain.pem /root/local/letsencrypt/live/grapplemma.com/fullchain.pem
```
8. Finally, restart the nignx server (while logged into Grapple's ec2 instance): `systemctl restart nginx`
9. Confirm the changes were successful by loading Grapple and checking the expiration and issue date on the certificate.
10. Done :)

# TODO
  Implement an automated CRON job to check if the certificate has reached 75% of its life, if it has then rotate it.

