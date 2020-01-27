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
    case class Peerster(val name: String) {
        var peers: List[String] = Nil
        var filters = List[String]()
        val uip = ui.next
        val gossipAddr = s"127.0.0.1:${gossip.next}"
        n.next
        def ps = if (peers.size > 0) s"-peers ${peers mkString ","}" else ""
        def fs = if(filters.size > 0) s"-filter ${filters mkString ","}" else ""
        def cmd = s"./peerster -name $name -UIPort $uip -gossipAddr $gossipAddr -N $n $ps $fs"
        def knows(peerster: Peerster): Peerster = {
            peers = peerster.gossipAddr :: peers
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
        def run(f: String => Unit): Future[String] = {
            println(s"$name running on $gossipAddr...")
            Future[String] {
                cmd ! ProcessLogger(f, err => println(s"$name panic-ed"))
                s"$name done"
            }
        }
        override def toString: String = cmd
    }
    //val result: Unit = "ls -al".!
    //val contents = Process("ls").lazyLines
    /*val alice = Peerster("Alice")
    val bob = Peerster("Bob")
    val charlie = Peerster("Charlie")
    val dave = Peerster("Dave")
    val eve = Peerster("Eve")
    val jack = Peerster("Jack")
    alice.knows(bob).knows(charlie).knows(eve)
    bob knows dave
    charlie knows dave
    eve knows jack
    jack knows dave
    val peersters = List(alice, bob, charlie, dave, eve, jack)
    val f = peersters map {case p: Peerster => p.run(line => println(s"${p.name}> $line"))}
    f foreach { posts =>
    for (post <- posts) println(post)
}*/
    val alice = Peerster("Alice")
    val bob = Peerster("Bob")
    val charlie = Peerster("Charlie")
    val dave = Peerster("Dave")
    val eve = Peerster("Eve")
    val jack = Peerster("Jack")
    val robert = Peerster("Robert")
    alice.knows(bob).knows(charlie)
    bob -> dave
    charlie -> dave
    eve -> jack
    jack -> dave
    val sub1 = alice ~> eve
    val sub2 = alice ~~~> robert
    alice - "init"
    def testAliceFindsOneProxy = {
        val peersters = List(alice, bob, charlie, dave, eve, jack, robert) ++ sub1 ++ sub2
        val aliceHasSendingTo: String => Unit = (line: String) => {
            //println(line)
            if(line contains "proxy") {
                println("[SUCCESS]")
                System exit 0
            }
        }
        val aliceHasMultipleProxies: String => Unit = {
            var cnt = 0
            (line: String) => {
                if (line contains "proxy") cnt += 1
                if(cnt >= 1) {
                    println("[SUCCESS]")
                    System exit 0
                }
            }
        }
        val ignore: String => Unit = (line: String) => {
            
        }
        //create map instead
        val testCases: List[String => Unit] = List(aliceHasMultipleProxies) ++ 1.until(peersters.size).map(i => ((s: String) => ignore(s)))
        //"go build" !
        //put timeout instead
        peersters.zip(testCases) map { case (p, fn) => p.run(fn) } foreach { case f => Await.result(f, Duration.Inf)}
    }
    testAliceFindsOneProxy
}