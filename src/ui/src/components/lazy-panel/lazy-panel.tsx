/*
 * Copyright 2018- The Pixie Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

import { buildClass } from 'app/utils/build-class';
import * as React from 'react';

import { makeStyles } from '@material-ui/core/styles';
import { createStyles } from '@material-ui/styles';

let called = false;
function triggerResize() {
  if (called) {
    return;
  }
  called = true;
  setTimeout(() => {
    window.dispatchEvent(new Event('resize'));
    called = false;
  });
}

interface LazyPanelProps {
  show: boolean;
  className?: string;
  children: React.ReactNode;
}

const useStyles = makeStyles(() => createStyles({
  panel: {
    '&:not(.visible)': {
      display: 'none',
    },
  },
}));

// LazyPanel is a component that renders the content lazily.
// eslint-disable-next-line react-memo/require-memo
export const LazyPanel: React.FC<LazyPanelProps> = ({ show, className, children }) => {
  const [rendered, setRendered] = React.useState(false);
  const classes = useStyles();

  React.useEffect(() => {
    setTimeout(triggerResize, 0);
  }, [show]);

  if (!show && !rendered) {
    return null;
  }
  if (show && !rendered) {
    setRendered(true);
  }

  return (
    <div className={buildClass(className, classes.panel, show && 'visible')}>
      {children}
    </div>
  );
};
LazyPanel.defaultProps = {
  className: '',
};
