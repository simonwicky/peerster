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
import { Network, Node, Edge } from 'react-vis-network'

function makeid(length) {
  var result           = '';
  var characters       = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
  var charactersLength = characters.length;
  for ( var i = 0; i < length; i++ ) {
     result += characters.charAt(Math.floor(Math.random() * charactersLength));
  }
  return result;
}

const Create = (props) => (
  /*
  <Nav className="mr-auto">
      <Nav.Link href="#home">Home</Nav.Link>
      <Nav.Link href="#link">Link</Nav.Link>
      <NavDropdown title="Dropdown" id="basic-nav-dropdown">
        <NavDropdown.Item href="#action/3.1">Action</NavDropdown.Item>
        <NavDropdown.Item href="#action/3.2">Another action</NavDropdown.Item>
        <NavDropdown.Item href="#action/3.3">Something</NavDropdown.Item>
        <NavDropdown.Divider />
        <NavDropdown.Item href="#action/3.4">Separated link</NavDropdown.Item>
      </NavDropdown>
    </Nav>
  */
    <Form inline>
      <FormControl type="text" placeholder="Name" className="mr-sm-2" onChange={e => props.update(e.target.value)} />
    </Form>
)

const Connect = (props) => (
  <InputGroup>
    <FormControl type="text" placeholder="Name"  />
    <InputGroup.Text><FontAwesomeIcon icon={faLongArrowAltRight} /></InputGroup.Text>
    <FormControl type="text" placeholder="Name"  />
  </InputGroup>
)

class Double extends React.Component {
  state = {
    a: '',
    b: '',
  }
  render = () => (
  <InputGroup>
    <FormControl type="text" placeholder="Name" value={this.state.a} onChange={e => {
      const val = e.target.value
      this.props.update([val, this.state.b])
      this.setState({a: val})
    }} />
    <InputGroup.Text><FontAwesomeIcon icon={this.props.icon} /></InputGroup.Text>
    <FormControl type="text" placeholder="Name" value={this.state.b} onChange={e => {
      const val = e.target.value
      this.props.update([this.state.a, val])
      this.setState({b: val})
    }} />
  </InputGroup>
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
      <Button variant="outline-secondary">
        <FontAwesomeIcon icon={faDatabase} />
        </Button>
      
  );
};

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
    input: null,
    nodes: [],
    edges: [],
    action: 'Generate',
  }
  connect = (nodeA, nodeB) => {
    const k = 2
    const n = 5 * k
    const nodes = this.state.nodes
    const set = new Set() // set of edges (to avoid self-loops and double edges)
    const used = new Set() // set of nodes (to know which nodes are zombies)
    const edges = this.state.edges
    const repr = (a, b) => (a < b) ? `${a}${b}` : `${b}${a}`
    const add = (a, b) => {
      if(!set.has(repr(a, b))) {
        edges.push([a, b])
        set.add(repr(a, b))
      } 
      used.add(a)
      used.add(b)
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
      add(nodes[j], nodes[next])
      j = next
      while(Math.random() < 0.4) {
        while(next == j) {
          next = Math.floor((Math.random() * (n-2*k-1)) + k + 1)
        }
        //edges.push([nodes[j], nodes[next]])
        add(nodes[j], nodes[next])
        j = next
      }
      //edges.push([nodes[next], nodes[n - i-1]])
      add(nodes[next], nodes[n - i-1])
    }
    for(var i = 0; i < k; ++i) {
      //edges.push(['Alice', nodes[i]])
      add(nodeA, nodes[i])
      //edges.push(['Bob', nodes[n-i-1]])
      add(nodeB, nodes[n-i-1])
    }
    //random walks
    this.setState({nodes: nodes.filter(node => used.has(node)), edges: edges})
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
  render = () => {
    console.log(this.state.edges)
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
              
            </SplitButton>
          </Navbar.Collapse>
        </Navbar>
        <Network style={{height: '600px'}}>
          {this.state.nodes.map(node => <Node label={node} decorator={Decorator} id={node} onClick={e => console.log(`${node} clicked`)}/>)}
          {this.state.edges.map(([u, v], i) => {
            return <Edge id={i} from={u}  to={v} />
          })}
        </Network>
      </div>
    )
  }
}

/*
<Node id="vader" label="Darth Vader" />
          <Node id="luke" label="Luke Skywalker" />
          <Node id="leia" label="Leia Organa" />
          <Edge id="1" from="vader" to="luke" />
          <Edge id="2" from="vader" to="leia" />
*/

export default App;
