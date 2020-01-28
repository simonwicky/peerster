import React from 'react';
import ReactDOM from 'react-dom'
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome'
import { faServer, faDatabase } from '@fortawesome/free-solid-svg-icons'
import 'bootstrap/dist/css/bootstrap.min.css'
import logo from './logo.svg';
import './App.css'
import 'react-bootstrap'
import CytoscapeComponent from 'react-cytoscapejs'
import Navbar from 'react-bootstrap/Navbar'
import Container from 'react-bootstrap/Container'
import Button from 'react-bootstrap/Button'
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
  state = {
    nodes: [],
    edges: [],
  }
  connect = (nodeA, nodeB) => {
    const k = 2
    const n = 5 * k
    const nodes = []
    const set = new Set()
    const used = new Set()
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
    const edges = []
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
    nodes.push(nodeA)
    nodes.push(nodeB)
    //random walks
    this.setState({nodes: nodes.filter(node => used.has(node)), edges: edges})
  }

  render = () => {
    console.log(this.state.edges)
    return (
      <div >
        <Navbar expand="lg" variant="light" bg="light">
          <Navbar.Brand href="#">Navbar</Navbar.Brand>
          <Navbar.Collapse>
            <Button onClick={e => this.connect('Alice', 'Bob')}>Gen</Button>
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
