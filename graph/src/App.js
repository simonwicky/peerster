import React from 'react';
import ReactDOM from 'react-dom'
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome'
import { faServer, faDatabase, faLongArrowAltRight } from '@fortawesome/free-solid-svg-icons'
import 'bootstrap/dist/css/bootstrap.min.css'
import logo from './logo.svg';
import './App.css'
import 'react-bootstrap'
import CytoscapeComponent from 'react-cytoscapejs'
import Navbar from 'react-bootstrap/Navbar'
import Container from 'react-bootstrap/Container'
import Button from 'react-bootstrap/Button'
import Dropdown from 'react-bootstrap/Dropdown'
import SplitButton from 'react-bootstrap/SplitButton'
import Form from 'react-bootstrap/Form'
import Col from 'react-bootstrap/Col'
import Row from 'react-bootstrap/Row'
import InputGroup from 'react-bootstrap/InputGroup'
import Nav from 'react-bootstrap/Nav'
import NavDropdown from 'react-bootstrap/NavDropdown'
import FormControl from 'react-bootstrap/FormControl'
import Toast from 'react-bootstrap/Toast'
import Card from 'react-bootstrap/Card'
import { Network, Node, Edge } from 'react-vis-network'
import Double from './components/Double'
import Create from './components/Create'
function makeid(length) {
  var result           = '';
  var characters       = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
  var charactersLength = characters.length;
  for ( var i = 0; i < length; i++ ) {
     result += characters.charAt(Math.floor(Math.random() * charactersLength));
  }
  return result;
}


class ProxyToasts extends React.Component {
  state = {
    proxies: new Map(),
  }
  componentDidUpdate = (prevProps, prevState, snapshot) => {
    const prev = new Set(prevProps.nodes)
    const updatables = this.props.nodes.filter(node => !prev.has(node))
    if(this.props.map) {
      updatables.forEach(updatable => {
        this.update(updatable)
        setTimeout(this.update(updatable), 3000)
      })
    }
    
  }
  update = updatable => () => {
    if (this.props.map.has(updatable)) {
      console.log(`http://localhost:${this.props.map.get(updatable).httpPort}/proxies`)
      fetch(`http://localhost:3000/proxies/${this.props.map.get(updatable).httpPort}`, {headers: { 'Content-Type': 'application/json', 'Accept': 'application/json', 'Access-Control-Allow-Origin': '*' },})
      .then(res => res.json())
      .then(proxies => {
        const nodesProxies = this.state.proxies
        nodesProxies.set(updatable, proxies)
        console.log(nodesProxies)
        this.setState({proxies: nodesProxies})
      }).catch(err => 'couldnt fetch proxies')
    }
    setTimeout(this.update(updatable), 3000)
  }
  repr = node => {
    if(this.state.proxies.has(node)) {
    return <Toast.Body>{this.state.proxies.get(node).proxies.map((proxy, i) => (
      <Card>
      <Card.Header>Proxy {i+1}</Card.Header>
      <Card.Body>
        <Card.Title>{proxy['IP']}</Card.Title>
        <Card.Text>
          <small>path_1</small> {proxy['_1']}
        </Card.Text>
        <Card.Text>
          <small>path_2</small> {proxy['_2']}
        </Card.Text>
      </Card.Body>
    </Card>
      ))}</Toast.Body>
    } else {
      return <Toast.Body>No proxies</Toast.Body>
    }
  }
  render = () => (
      <div
        aria-live="polite"
        aria-atomic="true"
        style={{
          minHeight: '200px', //positon relative
        }}
      >
        <div
          style={{
            position: 'absolute',
            top: 60,
            right: 16,
          }}
        >
          {this.props.nodes.map((node, i) => (
            <Toast onClose={e => this.props.untest(node)} style={{width: 300}} >
              <Toast.Header>
                <strong className="mr-auto">{node}</strong>
                <small>Proxies</small>
              </Toast.Header>
              {this.repr(node)}
            </Toast>
          ))}
        </div>
      </div>
  )
}


const CustomIcon = ({ icon, color = '#5596ed' }) => {
  const viewBox = 36;
  const iconSize = 20;
  const pad = (viewBox - iconSize) / 2;
  const center = viewBox / 2;
 
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox={`0 0 ${viewBox} ${viewBox}`}
    >
      <g>
        <circle cx={center} cy={center} r={16} fill={color} />
        <g transform={`translate(${pad}, ${pad})`}>
          {React.createElement(icon, { color: 'white', size: iconSize })}
        </g>
      </g>
    </svg>
  );
};
 
const Decorator = props => {
  return (
      <Button variant="outline-secondary" onClick={props.test}>
        <FontAwesomeIcon icon={faDatabase} />
        </Button>
      
  );
};

class Gen {
  constructor(i0) {
    this.i = i0
  }
  next = () => {
    ++this.i
    return this.i -1
  }
  get = () => this.i
}
let gossipPort = new Gen(5000)
let uiPort = new Gen(7000)
let httpPort = new Gen(8000)
let N = new Gen(0)
class Peerster {
  constructor(name) {
    this.name = name 
    this.gossipAddr = `127.0.0.1:${gossipPort.next()}`
    this.uiport = uiPort.next()
    this.httpPort = httpPort.next()
    this.peers = new Set()
    N.next()
  }
  peerStr = () => this.peers.size > 0 ? `-peers ${Array.from(this.peers).map(peer => peer.gossipAddr).join(',')}` : ''
  knows = (neighbour) => {
    this.peers.add(neighbour)
  }
  cmd = () => `./peerster -name ${this.name} -gossipAddr ${this.gossipAddr} -UIPort ${this.uiport} ${this.peerStr()} -N ${N.get()} -GUIPort ${this.httpPort}`
}
//id="dropdown-menu-align-right"
class App extends React.Component {
  actions = {
    'Create': <Create update={(name) => {
      this.setState({input: name})
    }}/>,
    'Connect': <Double icon={faLongArrowAltRight} update={(name) => {
      this.setState({input: name})
    }}/>,
    'Generate':<Double icon={faLongArrowAltRight} update={(name) => {
      this.setState({input: name})
    }}/>,
  }
  state = {
    tested: new Set(),
    input: null,
    nodes: [],
    edges: [],
    action: 'Create',
    objs: new Map(),
  }
  reset = () => {
    this.setState({
      tested: new Set(),
      input: null,
      nodes: [],
      edges: [],
      objs: new Map(),
    })
  }
  connect = (nodeA, nodeB) => {
    const k = 3
    const n = 5 * k
    const nodes = []
    const set = new Set() // set of edges (to avoid self-loops and double edges)
    const used = new Set() // set of nodes (to know which nodes are zombies)
    const edges = this.state.edges
    const repr = (a, b) => (a < b) ? `${a}${b}` : `${b}${a}`
    const add = (a, b) => {
      if(!set.has(repr(a, b)) && a != b) {
        edges.push([a, b])
        set.add(repr(a, b))
        used.add(a)
        used.add(b)
        return true
      } 
        return false
    }
    for(var i = 0; i < n; ++i) {
      nodes.push(makeid(5))
    }
    /*for(var i = 4*k; i < n-k; ++i) {
      nodes.push(makeid(5))
    }*/
    for(var i = 0; i <= k; ++i) {
      let j = i
      let next = Math.floor((Math.random() * (n-2*k-1)) + k + 1)
      //edges.push([nodes[j], nodes[next]])
      while(!add(nodes[j], nodes[next])) {
        next = Math.floor((Math.random() * (n-2*k-1)) + k + 1)
      }
      j = next
      while(Math.random() < 0.4) {
        while(next == j) {
          next = Math.floor((Math.random() * (n-2*k-1)) + k + 1)
        }
        //edges.push([nodes[j], nodes[next]])
        if (add(nodes[j], nodes[next])) {
          j = next
        }
      }
      //edges.push([nodes[next], nodes[n - i-1]])
      add(nodes[next], nodes[n - i-1])
    }
    //connect the ndes to the start nodes
    for(var i = 0; i <= k; ++i) {
      //edges.push(['Alice', nodes[i]])
      add(nodeA, nodes[i])
      //edges.push(['Bob', nodes[n-i-1]])
      add(nodeB, nodes[n-i-1])
    }
    //random walks
    this.setState({nodes: this.state.nodes.concat(nodes.filter(node => used.has(node))), edges: edges})
  }
  do = () => {
    switch (this.state.action) {
      case 'Create':
        const nodes = this.state.nodes
        nodes.push(this.state.input)
        this.setState({nodes: nodes})
        break
      case 'Connect': 
      console.log('connecting..')
        const [cA, cB] = this.state.input
        const edges = this.state.edges
        edges.push([cA, cB])
        this.setState({edges: edges})
        break
      case 'Generate':
        const [gA, gB] = this.state.input
        this.connect(gA, gB)
        break
    }
  }
  run = () => {
    const nodes = new Map()
    this.state.nodes.forEach(node => {
      nodes.set(node, new Peerster(node))
    })
    this.state.edges.forEach(([u, v]) => nodes.get(u).knows(nodes.get(v)))
    const cmds = []
    nodes.forEach(v => cmds.push(v.cmd()))
    console.log(cmds)
    const query = JSON.stringify({cmds:cmds})
    fetch('http://localhost:3000/', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: query,
    })
    this.setState({objs: nodes})
  }
  test = node => () => {
    const tested = this.state.tested
    tested.add(node)
    this.setState({tested: tested})
  }
  untest = node => {
    const tested = this.state.tested
    tested.delete(node)
    this.setState({tested: tested})
  }
  render = () => {
    //<Button onClick={e => this.connect('Alice', 'Bob')}>Gen</Button>
    return (
      <div >
        <Navbar expand="lg" variant="light" bg="light" className="justify-content-between">
          <Navbar.Brand href="#">Peerster</Navbar.Brand>
          
          <Navbar.Toggle />
          <Navbar.Collapse className="justify-content-end">
          {this.actions[this.state.action]}
            <SplitButton
              title={this.state.action}
              variant='primary'
              id={`dropdown-split-variants-primary`}
              key='primary'
              drop='left'
              onClick={this.do}
            >
              {['Create', 'Connect', 'Generate'].map((act, i) => <Dropdown.Item onClick={e => this.setState({action: act})} eventKey={i}>{act}</Dropdown.Item>)}
              <Dropdown.Divider />
              <Dropdown.Item onClick={this.reset}>Reset</Dropdown.Item>
            </SplitButton>
            <Button variant="danger" onClick={this.run}>Run</Button>
          </Navbar.Collapse>
        </Navbar>
        <Network style={{height: '800px'}}>
          {this.state.nodes.map(node => <Node label={node} decorator={() => <Decorator test={this.test(node)}/>} id={node} />)}
          {this.state.edges.map(([u, v], i) => {
            return <Edge id={i} from={u}  to={v} />
          })}
        </Network>
        <ProxyToasts nodes={[...this.state.tested]} untest={this.untest} map={this.state.objs} />
      </div>
    )
  }
}

export default App;
