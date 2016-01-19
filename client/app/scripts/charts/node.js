import React from 'react';
import ReactDOM from 'react-dom';
import d3 from 'd3';
import { Motion, spring } from 'react-motion';

import { clickNode, enterNode, leaveNode } from '../actions/app-actions';
import { getNodeColor } from '../utils/color-utils';

import PureRenderMixin from 'react-addons-pure-render-mixin';
import reactMixin from 'react-mixin';

export default class Node extends React.Component {
  constructor(props, context) {
    super(props, context);
    this.handleMouseClick = this.handleMouseClick.bind(this);
    this.handleMouseEnter = this.handleMouseEnter.bind(this);
    this.handleMouseLeave = this.handleMouseLeave.bind(this);
  }

  render() {
    const props = this.props;
    const nodeScale = props.focused ? props.selectedNodeScale : props.nodeScale;
    const zoomScale = this.props.zoomScale;
    let scaleFactor = 1;
    if (props.focused) {
      scaleFactor = 1.25 / zoomScale;
    } else if (props.blurred) {
      scaleFactor = 0.75;
    }
    let labelOffsetY = 18;
    let subLabelOffsetY = 35;
    const isPseudo = !!this.props.pseudo;
    const color = isPseudo ? '' : getNodeColor(this.props.rank, this.props.label);
    const onMouseEnter = this.handleMouseEnter;
    const onMouseLeave = this.handleMouseLeave;
    const onMouseClick = this.handleMouseClick;
    const classNames = ['node'];
    const animConfig = [80, 20]; // stiffness, bounce
    const label = this.ellipsis(props.label, 14, nodeScale(4 * scaleFactor));
    const subLabel = this.ellipsis(props.subLabel, 12, nodeScale(4 * scaleFactor));
    let labelFontSize = 14;
    let subLabelFontSize = 12;

    if (props.focused) {
      labelFontSize /= zoomScale;
      subLabelFontSize /= zoomScale;
      labelOffsetY /= zoomScale;
      subLabelOffsetY /= zoomScale;
    }
    if (this.props.highlighted) {
      classNames.push('highlighted');
    }
    if (this.props.blurred) {
      classNames.push('blurred');
    }
    if (this.props.pseudo) {
      classNames.push('pseudo');
    }
    const classes = classNames.join(' ');

    return (
      <Motion style={{
        x: spring(this.props.dx, animConfig),
        y: spring(this.props.dy, animConfig),
        f: spring(scaleFactor, animConfig),
        labelFontSize: spring(labelFontSize, animConfig),
        subLabelFontSize: spring(subLabelFontSize, animConfig),
        labelOffsetY: spring(labelOffsetY, animConfig),
        subLabelOffsetY: spring(subLabelOffsetY, animConfig)
      }}>
        {function(interpolated) {
          const transform = `translate(${d3.round(interpolated.x, 2)},${d3.round(interpolated.y, 2)})`;
          return (
            <g className={classes} transform={transform} id={props.id}
              onClick={onMouseClick} onMouseEnter={onMouseEnter} onMouseLeave={onMouseLeave}>
              {props.highlighted && <circle r={nodeScale(0.7 * interpolated.f)} className="highlighted"></circle>}
              <circle r={nodeScale(0.5 * interpolated.f)} className="border" stroke={color}></circle>
              <circle r={nodeScale(0.45 * interpolated.f)} className="shadow"></circle>
              <circle r={Math.max(2, nodeScale(0.125 * interpolated.f))} className="node"></circle>
              <text className="node-label" textAnchor="middle" style={{fontSize: interpolated.labelFontSize}}
                x="0" y={interpolated.labelOffsetY + nodeScale(0.5 * interpolated.f)}>
                {label}
              </text>
              <text className="node-sublabel" textAnchor="middle" style={{fontSize: interpolated.subLabelFontSize}}
                x="0" y={interpolated.subLabelOffsetY + nodeScale(0.5 * interpolated.f)}>
                {subLabel}
              </text>
            </g>
          );
        }}
      </Motion>
    );
  }

  ellipsis(text, fontSize, maxWidth) {
    const averageCharLength = fontSize / 1.5;
    const allowedChars = maxWidth / averageCharLength;
    let truncatedText = text;
    if (text && text.length > allowedChars) {
      truncatedText = text.slice(0, allowedChars) + '...';
    }
    return truncatedText;
  }

  handleMouseClick(ev) {
    ev.stopPropagation();
    clickNode(this.props.id, this.props.label, ReactDOM.findDOMNode(this).getBoundingClientRect());
  }

  handleMouseEnter() {
    enterNode(this.props.id);
  }

  handleMouseLeave() {
    leaveNode(this.props.id);
  }
}

reactMixin(Node.prototype, PureRenderMixin);
