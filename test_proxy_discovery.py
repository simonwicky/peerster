import subprocess
from functools import reduce
import signal
import sys
gossipPort = 6000
UIPort = 7000
N = 0
peersters = []
class Peerster:
    def __init__(self, name):
        super().__init__()
        global UIPort
        global N
        global peersters
        global gossipPort
        self.name = name
        self.UIPort = UIPort
        self.gossipPort = gossipPort
        self.peers = []
        N += 1
        UIPort += 1
        gossipPort += 1
        peersters += [self]
    def knows(self, that):
        self.peers += ['127.0.0.1:{}'.format(that.gossipPort)]
    def run(self):
        global N
        peers = ['-peers', reduce(lambda a, b: a+','+str(b), self.peers)] if len(self.peers) > 0 else []
        base = ['./Peerster', '-name', self.name]
        ui = ['-UIPort', str(self.UIPort)]
        n = ['-N', str(N)]
        gossip = ['-gossipAddr', '127.0.0.1:{}'.format(self.gossipPort)]
        process = subprocess.Popen(base+gossip+ui+peers+n)
        return process

if __name__ == '__main__':
    subprocess.call(['go', 'build'])
    alice = Peerster('Alice')
    bob = Peerster('Bob')
    charlie = Peerster('Charlie')
    dave = Peerster('Dave')
    eve = Peerster('Eve')
    fred = Peerster('Fred')
    gerry = Peerster('Gerry')
    alice.knows(charlie)
    alice.knows(bob)
    alice.knows(fred)
    alice.knows(gerry)
    bob.knows(dave)
    charlie.knows(dave)
    processes = []
    for peerster in peersters:
        processes += [peerster.run()]
    def signal_handler(sig, frame):
        for process in processes:
            process.kill()
        subprocess.call(['pkill', 'Peerster'])
        sys.exit(0)
    signal.signal(signal.SIGINT, signal_handler)