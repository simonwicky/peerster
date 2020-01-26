import subprocess
from functools import reduce
import signal
import sys
import time
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
        self.muted = False
        self.process = None
        N += 1
        UIPort += 1
        gossipPort += 1
        peersters += [self]
    def knows(self, that):
        self.peers += ['127.0.0.1:{}'.format(that.gossipPort)]
        return self
    def mute(self):
        self.muted = True
        return self
    def run(self):
        global N
        peers = ['-peers', reduce(lambda a, b: a+','+str(b), self.peers)] if len(self.peers) > 0 else []
        base = ['./Peerster', '-name', self.name]
        ui = ['-UIPort', str(self.UIPort)]
        n = ['-N', str(N)]
        gossip = ['-gossipAddr', '127.0.0.1:{}'.format(self.gossipPort)]
        cmd = base+gossip+ui+peers+n
        print(reduce(lambda x,y: str(x) + ' '+str(y), cmd))
        #process = subprocess.Popen(cmd, stdout=subprocess.PIPE) if self.muted else subprocess.Popen(base+gossip+ui+peers+n)
        process = None
        self.process = process
        return process
    def dump(self):
        print(self.name, self.process)
        self.process.kill()
        return self.process.stdout.read()

if __name__ == '__main__':
    subprocess.call(['go', 'build'])
    alice = Peerster('Alice')
    bob = Peerster('Bob')
    charlie = Peerster('Charlie')
    dave = Peerster('Dave')
    eve = Peerster('Eve')
    #fred = Peerster('Fred').mute()
    #gerry = Peerster('Gerry').mute()
    alice.knows(charlie).knows(bob).knows(eve)#.knows(fred).knows(gerry)
    bob.knows(dave)
    charlie.knows(dave)
    processes = [peerster.run() for peerster in peersters[:5]]
    time.sleep(30)
    for process in processes:
        process.kill
    