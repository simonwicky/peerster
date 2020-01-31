//Author: Frédéric Gessler
import scala.language.postfixOps
import sys.process._
import scala.concurrent._
import ExecutionContext.Implicits.global
import scala.concurrent.duration._
import scala.util.Random
object Hello extends App {
    case class RandomIDGen() {
        val r = scala.util.Random
        //var namesDict = 0.to()
        //List("Theresa", "Omar", "Sandra", "Jonny", "Toni", "Claudia", )
        def next: String = {
            0.to(r.nextInt(10)).map(i => r.nextPrintableChar).mkString("")
            //val name :: tail = namesDict
            //namesDict = tail
            //name
        }
    }
    val names = RandomIDGen()
    case class Counter(init: Int) {
        var i = init
        def next: Int = {
            i += 1
            i - 1
        }
        override def toString = i.toString
    }
    val gossip = Counter(6001)
    val ui = Counter(7000)
    var n = Counter(0)
    val ignore: String => Unit = (line: String) => {
            
        }
    case class Peerster(val name: String) {
        var peers: Set[String] = Set()
        var filters = List[String]()
        val uip = ui.next
        val gossipAddr = s"127.0.0.1:${gossip.next}"
        n.next
        def ps = if (peers.size > 0) s"-peers ${peers mkString ","}" else ""
        def fs = if(filters.size > 0) s"-filter ${filters mkString ","}" else ""
        def cmd = s"./peerster -name $name -UIPort $uip -gossipAddr $gossipAddr -N $n $ps $fs"
        def knows(peerster: Peerster): Peerster = {
            peers = peers + peerster.gossipAddr
            this
        }
        def -(filter: String) = {
            filters = filter :: filters
        }
        def ->(peerster: Peerster): Peerster = knows(peerster)
        /*def <-(peerster: Peerster): Peerster = peerster -> this
        def <->(peerster: Peerster) :Peerster = {
            this <- peerster
            this -> peerster
        }*/
        def ~>(peerster: Peerster): List[Peerster] = {
            val nodeA = Peerster(names.next)
            val nodeB = Peerster(names.next)
            this -> nodeA
            this -> nodeB
            nodeA -> peerster
            nodeB -> peerster
            nodeA :: nodeB :: Nil
        }
        def ~~~>(peerster: Peerster): List[Peerster] = {
            val k = 4
            val n = 5*k
            //procedurally generate a sub network between the 2 nodes
            //generate n random nodes; do k random walks
            val subNetwork = 0.until(n).map(_ => Peerster(names.next))
            val r = scala.util.Random
            for(i <- 0.until(k)) {
                this -> subNetwork(i)
                var j = i
                while (r.nextFloat < 0.6) {
                    val next = r.nextInt(n - 2*k) + k
                    subNetwork(j) -> subNetwork(next)
                    j = next
                }
                subNetwork(j) -> peerster
            }
            subNetwork.toList
        }
        def run(f: String => String => Unit): Future[String] = {
            println(s"$cmd")
            Future[String] {
                val fn = f(name)
                cmd ! ProcessLogger(fn, err => if(err contains "panic:") println(Console.RED+"$"+name+Console.WHITE+s"> $err"))
                s"$name done"
            }
        }
        override def toString: String = cmd
    }
    val alice = Peerster("Alice")
    val bob = Peerster("Bob")
    val charlie = Peerster("Charlie")
    val dave = Peerster("Dave")
    val eve = Peerster("Eve")
    val jack = Peerster("Jack")
    val robert = Peerster("Robert")
    val amandine = Peerster("Amandine")
    /*alice.knows(bob).knows(charlie)
    bob -> dave
    charlie -> dave
    eve -> jack
    jack -> dave
    val sub1 = alice ~> eve
    val sub2 = alice ~~~> robert
    val sub3 = alice ~~~> amandine
    amandine -> robert
    alice - "init"
    alice - "series"
    alice - "fwd"
    alice - "rec"
    def testAliceFindsOneProxy = {
        val peersters = List(alice, bob, charlie, dave, eve, jack, robert) ++ sub1 ++ sub2 ++ sub3
        val aliceHasSendingTo: String => String => Unit = (name: String) => (line: String) => {
            if(line contains "proxy") {
                println("[SUCCESS]")
                System exit 0
            }
        }
        val hasMultipleProxies: String => String => Unit = (name: String) => {
            var cnt = 0
            (line: String) => {
                if (line contains "FATAL") println(line)
                if (line contains "proxy") {
                    println("one proxy added!")
                    cnt += 1
                }
                if(cnt >= 1) {
                    println("[SUCCESS]")
                    System exit 0
                }
            }
        }
        
        val printFatals: String => String => Unit = (name: String) => (line: String) => {
            if (line.contains( "FATAL" )|| line.contains( "WARN")) {
                println(s"$name> $line")
            }
        }
        //create map instead
        val testCases: List[String => String => Unit] = List(hasMultipleProxies) ++ 1.until(peersters.size)
        .map(i => ((s: String) => printFatals(s)))
        //"go build" !
        //put timeout instead
        peersters.zip(testCases) map { case (p, fn) => p.run(fn) } foreach { case f => Await.result(f, Duration.Inf)}
    }
    
    "go build"!ProcessLogger(ignore, ignore)
    testAliceFindsOneProxy*/
    alice run ((x: String) => (y: String) => {})
}