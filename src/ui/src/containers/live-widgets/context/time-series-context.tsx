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

import * as React from 'react';

type Domain = [number, number];
type DomainFn = ((domain: Domain) => Domain);

interface TimeSeriesContextProps {
  hoverTime: number | null;
  setHoverTime: (time: number) => void;
  setTimeseriesDomain: (domain: Domain | DomainFn) => void;
  timeseriesDomain: Domain | null;
}

export const TimeSeriesContext = React.createContext<TimeSeriesContextProps>(null);

export const TimeSeriesContextProvider: React.FC = ({ children }) => {
  const [hoverTime, setHoverTime] = React.useState<number | null>(null);
  const [timeseriesDomain, setTimeseriesDomain] = React.useState<Domain | null>(null);

  const context = React.useMemo(() => ({
    hoverTime,
    setHoverTime,
    timeseriesDomain,
    setTimeseriesDomain,
  }), [hoverTime, setHoverTime, timeseriesDomain, setTimeseriesDomain]);

  return (
    <TimeSeriesContext.Provider value={context}>
      {children}
    </TimeSeriesContext.Provider>
  );
};

export function withTimeSeriesContext<P>(Component: React.ComponentType<P>): React.ComponentType<P> {
  return React.memo(function TimeSeriesContextProviderHOC(props) {
    return (
      <TimeSeriesContextProvider>
        <Component {...props} />
      </TimeSeriesContextProvider>
    );
  });
}
