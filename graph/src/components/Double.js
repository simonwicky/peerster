import React from 'react'
import InputGroup from 'react-bootstrap/InputGroup'
import FormControl from 'react-bootstrap/FormControl'
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome'

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
export default Double  