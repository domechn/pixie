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

import {
  gql, useQuery, useMutation, useLazyQuery,
} from '@apollo/client';
import Table from '@material-ui/core/Table';
import TableBody from '@material-ui/core/TableBody';
import TableHead from '@material-ui/core/TableHead';
import TableRow from '@material-ui/core/TableRow';
import { distanceInWords } from 'date-fns';
import * as React from 'react';

import { GQLDeploymentKey, GQLDeploymentKeyMetadata } from 'app/types/schema';
import {
  AdminTooltip, StyledTableCell, StyledTableHeaderCell,
  StyledLeftTableCell,
} from './utils';
import {
  useKeyListStyles, KeyActionButtons,
} from './key-list';
import { MonoSpaceCell } from './cluster-table-cells';

interface DeploymentKeyDisplay {
  id: string;
  idShort: string;
  createdAt: string;
  desc: string;
}

export function formatDeploymentKey(depKey: GQLDeploymentKeyMetadata): DeploymentKeyDisplay {
  const now = new Date();
  return {
    id: depKey.id,
    idShort: depKey.id.split('-').pop(),
    createdAt: `${distanceInWords(new Date(depKey.createdAtMs), now, { addSuffix: false })} ago`,
    desc: depKey.desc,
  };
}

export const DeploymentKeyRow = React.memo<{
  deploymentKey: DeploymentKeyDisplay
}>(function DeploymentKeyRow({ deploymentKey }) {
  const [deleteDeploymentKey] = useMutation<{ DeleteDeploymentKey: boolean }, { id: string }>(gql`
    mutation DeleteDeploymentKeyFromAdminPage($id: ID!) {
      DeleteDeploymentKey(id: $id)
    }
  `);

  const [copyKeyToClipboard] = useLazyQuery<{ deploymentKey: GQLDeploymentKey }>(gql`
    query getDeploymentKey($id:ID!){
      deploymentKey(id: $id) {
        id
        key
      }
    }`, {
    fetchPolicy: 'network-only',
    onCompleted: (data) => {
      navigator.clipboard.writeText(data?.deploymentKey?.key).then();
    },
  });

  const copyAction = React.useCallback(() => {
    copyKeyToClipboard({ variables: { id: deploymentKey.id } });
  }, [deploymentKey.id, copyKeyToClipboard]);

  const deleteAction = React.useCallback(() => {
    deleteDeploymentKey({
      variables: { id: deploymentKey.id },
      update: (cache, { data }) => {
        if (!data.DeleteDeploymentKey) {
          return;
        }
        cache.modify({
          fields: {
            deploymentKeys(existingKeys, { readField }) {
              return existingKeys.filter(
                (key) => (deploymentKey.id !== readField('id', key)),
              );
            },
          },
        });
      },
      optimisticResponse: { DeleteDeploymentKey: true },
    }).then();
  }, [deploymentKey.id, deleteDeploymentKey]);

  return (
    <TableRow key={deploymentKey.id}>
      <AdminTooltip title={deploymentKey.id}>
        <StyledLeftTableCell>{deploymentKey.idShort}</StyledLeftTableCell>
      </AdminTooltip>
      <StyledTableCell>{deploymentKey.createdAt}</StyledTableCell>
      <StyledTableCell>{deploymentKey.desc}</StyledTableCell>
      <MonoSpaceCell data={'••••••••••••'} />
      <StyledTableCell>
        <KeyActionButtons
          copyOnClick={copyAction}
          deleteOnClick={deleteAction} />
      </StyledTableCell>
    </TableRow>
  );
});

export const DeploymentKeysTable = React.memo(function DeploymentKeysTable() {
  const classes = useKeyListStyles();
  const { data, error } = useQuery<{ deploymentKeys: GQLDeploymentKey[] }>(
    gql`
      query getDeploymentKeysForAdminPage{
        deploymentKeys {
          id
          desc
          createdAtMs
        }
      }
    `,
    // As deployment keys can be sensitive, skip cache on first fetch.
    { pollInterval: 60000, fetchPolicy: 'network-only', nextFetchPolicy: 'cache-and-network' },
  );

  if (error) {
    return <div className={classes.error}>{error.toString()}</div>;
  }

  const deploymentKeys = (data?.deploymentKeys ?? []).map((key) => formatDeploymentKey(key));
  return (
    <>
      <Table>
        <TableHead>
          <TableRow>
            <StyledTableHeaderCell>ID</StyledTableHeaderCell>
            <StyledTableHeaderCell>Created</StyledTableHeaderCell>
            <StyledTableHeaderCell>Description</StyledTableHeaderCell>
            <StyledTableHeaderCell>Value</StyledTableHeaderCell>
            <StyledTableHeaderCell />
          </TableRow>
        </TableHead>
        <TableBody>
          {deploymentKeys.map((deploymentKey: DeploymentKeyDisplay) => (
            <DeploymentKeyRow key={deploymentKey.id} deploymentKey={deploymentKey} />
          ))}
        </TableBody>
      </Table>
    </>
  );
});
