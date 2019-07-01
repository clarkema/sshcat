# sshcat

`sshcat` is a simple tool analogous to socat, but using SSH.

The original motivation for it was to provide a straightforward means of
transferring large files between two users on different Unix-style machines
without both users having to have an account on the same machine.

Sending large files (read: streams of bytes) is still
[far more trouble than you would hope](https://xkcd.com/949/).

Assuming that one of the parties to the transfer can bind to an interface with
a global IP address (either directly or indirectly via an SSH tunnel or
port-forwarding through NAT) they could use
[socat](http://www.dest-unreach.org/socat/):

Sender: `socat OPEN:hugefile,rdonly TCP4-LISTEN:2222`

Receiver: `socat TCP4:serverip:2222 - > hugefile`

There are various permutations of possible commands depending on which side of
the transfer can bind to a global interface and who has the file.

If you don’t care about authentication or encryption of the data in flight,
this works fine.  If you _do_ care... well, it gets a bit messier.  Socat does
offer SSL support, and you could wrap it with some scripts to generate the
required certificates to authenticate client and server to each other and
protect the data in flight, but that starts to become a hassle.

SSH offers an easy way to move encrypted streams of bytes between hosts - the
only reason not to use it for this task is the desire to avoid having to
set up a temporary account for one of the parties on whichever machine is
easiest to access; securing that account; and then tearing it down when the
transfer is complete.

`sshcat` provides a dumb, temporary SSH server that has no links to any user
database, no login capability, and none of the other subsystems (including
SFTP) offered by a normal SSH server.

All it does is connect the `STDIN` and `STDOUT` of the ssh client with those of
`sshcat` itself.  That’s it.  Simple as it is though, it’s still enough to
enable real-time file transfers between two parties who have some other
"secure enough" means of communication to co-ordinate the transfer, which
turns out to be quite useful.

## Examples

Alice wants to send a file to Bob.  Alice can bind to a global interface, so she
starts `sshcat` while Bob uses standard `ssh` to receive:

Alice: `sshcat --password FOO < hugefile`

Bob: `ssh -Tn -p 2222 serverip > hugefile`

Alice wants to send a file to Bob.  _Bob_ can bind to a global interface,
but Alice cannot, so this time Bob starts `sshcat` in receive mode while
Alice uses standard `ssh` to send:

Bob: `sshcat --password FOO < /dev/null > hugefile`

Alice: `ssh -T -p 2222 serverip < hugefile`

**NOTE** in both cases it is important to ensure that the side _not_ sending
data closes their `STDIN`; otherwise the connection will hang and stay open
even after the tranfer is complete.  When receiving with `ssh` this is
achieved with the `-n` flag.  When receiving with `sshcat`, use `< /dev/null`

## Other uses

The simplicity of sshcat means it fits well into a wider toolbox; it can also be
combined with many other socat commands to add an ad-hoc SSH account to the
mix.

## Caveats and security considerations

 * Password are specified with the `--password` flag; these will be visible
   in the process list of the machine running `sshcat`.  Key-based
   authentication is planed but not yet implemented.

 * sshcat currently generates a new host key on each run, and doesn’t display
   it for confirmation.  Improvements on this front are planned.

 * sshcat’s flexibility means it could be used to expose all sorts of
   processes (up to and including shells) via an ad-hoc SSH interface.

   Be extremely cautious with this.  It might sound like a good way of
   providing access to a specific process on your machine, but many tools you
   might not expect offer some kind of escape mechanism which would end up
   granting full shell access.
