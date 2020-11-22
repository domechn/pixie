import * as React from 'react';

import { FrameElement } from 'utils/frame-utils';
import { AuthMessageBox } from './message';

export default {
  title: 'Auth/Message Box',
  component: AuthMessageBox,
  decorators: [
    (Story) => (
      <FrameElement width={500}>
        <Story />
      </FrameElement>
    ),
  ],
};

export const Completed = () => (
  <AuthMessageBox
    title='Auth Completed'
    message='Please close this window and return to the CLI.'
  />
);

export const Error = () => (
  <AuthMessageBox
    error='recoverable'
    title='Auth Failed'
    message='Check your spelling and try again.'
  />
);

export const ErrorDetails = () => (
  <AuthMessageBox
    error='fatal'
    errorDetails='Internal error: bad things happened'
    title='Auth Failed'
    message='Login to this org is not allowed.'
  />
);

export const Code = () => (
  <AuthMessageBox
    title='Code Box'
    message='Please copy and paste this code!'
    code='a9123sd12321asda-sd123213as-as12'
  />
);
