import React from 'react';
import {Box} from "grommet";

export default () =>
    <Box fill="vertical" overflow="auto">
        <h3>The Nature of the EdgeNet Testbed</h3>
        <p>
            EdgeNet consists of computational resources hosted by organizations
            (principally research organizations like universities) that donate their own time, rack space,
            and network connectivity for the good of the community. As an overlay, EdgeNet is not a
            "testbed" in the usual sense of a controlled environment for experiments.
        </p>
        <p>
            It provides access to other testbeds that are interconnected with EdgeNet in a worldwide federation.
            It also allows the deployment of experimental services that are accessible to all users of the internet.
        </p>
        <p>
            Running an experiment on EdgeNet is fundamentally different from running it in a LAN-based
            lab or on an isolated wide-area testbed.
        </p>
        <p>
            All aspects of this policy governing the use of the EdgeNet testbed apply equally to the User
            Member’s use of other testbeds that are federated with EdgeNet and that it accesses through
            the EdgeNet testbed.
        </p>
        <p>
            If those testbeds have additional requirements, User Member's access to those testbeds will be conditioned
            upon agreement to those requirements, in a separate agreement.
        </p>
        <br />

        <h3>General Guidance on Experiments</h3>
        <p>
            A good litmus test when considering whether an experiment is appropriate for EdgeNet is to
            ask what the network administrator at your organization would say about the experiment running on your local site.
        </p>
        <p>
            If the experiment disrupts local activity (e.g., uses more than its share of your site’s internet bandwidth)
            or triggers complaints from remote network administrators (e.g., performs systematic port scans), then it is not
            appropriate for EdgeNet.
        </p>
        <p>
            It is your responsibility to ensure that your use of EdgeNet falls within these constraints.
            This means that you should debug your code in a controlled environment so that you have confidence that
            you understand its behavior.
        </p>

        <h3>Responsibility of Sites with Regard to Their Users</h3>
        <p>
            EdgeNet is designed to support a broad community of end-users, both User Members of EdgeNet
            and users of the federated testbeds, as well as external end-users who access experimental services
            that are deployed on the testbed.
        </p>
        <p>
            As a consequence, EdgeNet could indirectly support users that have not officially registered with
            EdgeNet, and may even be unknown to you (the resource provider).
        </p>
        <p>
            It is your responsibility as a site administrator to ensure that your users do not cause your service to
            violate the terms of this Acceptable Use Policy. In particular, site administrators should ensure that
            their users are not able to hijack the service and use it to attack or spam other nodes or network users.
        </p>

        <h3>Standards of Network Etiquette</h3>
        <p>
            EdgeNet is designed to support network measurement experiments that purposely probe the Internet.
            However, we expect all users to adhere to widely-accepted standards of network etiquette in an effort to
            minimize complaints from network administrators.
        </p>
        <p>
            Activities that have been interpreted as worm and denial-of-service attacks in the past (and should be avoided)
            include sending SYN packets to port 80 on random machines, probing random IP addresses, repeatedly pinging
            routers, overloading bottleneck links with measurement traffic, and probing a single target machine from
            many EdgeNet nodes.</p>
        <p>
            It is likely that individual EdgeNet Sites will have their own Acceptable Use Policies. Users should
            not knowingly violate such local Acceptable Use Policies. Conflicts between Site Acceptable Use Policies
            and EdgeNet's stated goal of supporting research into wide-area networks should be brought to the
            attention of EdgeNet administrators.</p>
        <p>
            The expectations placed on EdgeNet Sites are described in a companion document:
            <a href="https://planet-lab.eu/HostingResponsibilities">Hosting Responsibilities.</a>
        </p>

        <h3>Handling Complaints</h3>
        <p>
            While the central EdgeNet authority is often the first point-of-contact for complaints about
            misbehaving services, it is our policy to put the complainant in direct contact with the researcher
            who is responsible for the service.</p>
        <p>
            To report a suspected violation contact: <a href="mailto:support@planet-lab.eu">EdgeNet Support</a>.
        </p>
        <h3>No Guarantees</h3>
        <ul>
            <li>
                EdgeNet provides absolutely no privacy guarantees with regard to packets sent to/from
                slices (a “slice” being a set of virtual machines and other resources obtained by a user from EdgeNet
                 and its federated testbeds).
            </li>
            <li>
                Users should assume packets will be monitored and logged, for example, to allow other users to investigate
                abuse (see previous paragraph).
            </li>
            <li>
                EdgeNet also does not provide any guarantees with respect to the reliability of individual nodes,
                which may be rebooted or reinstalled any time. Reinstalling a node implies that the disk is wiped, meaning
                that users should not treat the local disk as a persistent form of storage.
            </li>
            <li>
                Any goods, services, and written materials provided by EdgeNet or its agents or any member in any
                form, whether furnished in draft or final form are provided "as-is with all defects" and without any warranty
                of any kind. EdgeNet disclaims all warranties of merchantability, fitness for a particular purpose,
                and noninfringment.
            </li>
            <li>
                In no event shall EdgeNet or any other member be liable to any other member of EdgeNet for
                any consequential, incidental, punitive or lost profit damages, or for any damages arising out of loss of use
                or loss of data, to the extent that such damages arise out of the activities of EdgeNet or this
                agreement or any breach thereof even if member has been advised of the possibility of such damages.
            </li>
        </ul>
        <p>
            Nothing contained in this Agreement shall be deemed as creating any rights or liabilities in or for third parties
            who are not Members of EdgeNet.
        </p>
        <h3>Rules of Use</h3>
        <ul>
            <li><strong>Overall Rules</strong></li>
            <ul>
                <li>
                    EdgeNet should not be used for any illegal enacted by any Law or Regulation.
                </li>
                <li>
                    EdgeNet should not be used for any commercial activities. Use for research and educational
                    purposes is allowed.
                </li>
                <li>
                    Access rights granted to EdgeNet exclude any rights to sublicense, including to affiliates,
                    unless expressly stated otherwise.
                </li>
                <li>
                    Access rights granted to EdgeNet don’t give the rights to accede to any other
                    platform that is not federated with EdgeNet.
                </li>
                <li>
                    While EdgeNet is federated with other testbeds, access rights to those testbeds may be
                    restricted by those testbeds or by agreements between EdgeNet and those testbeds.
                </li>
            </ul>
            <li><strong>Node Usage Rules</strong></li>
            <ul>
                <li>
                    Use existing security mechanisms. For example, all access to EdgeNet nodes must be via SSH.
                </li>
                <li>
                    Do not circumvent accounting and auditing mechanisms. This means you must associate your identity
                    with the EdgeNet slice (account) in which your service runs. You must not do anything to
                    obfuscate the audit trail.
                </li>
                <li>
                    No hacking attempts of the EdgeNet nodes. This includes "red team" (hacker test) experiments.
                    All access is non-root.
                </li>
                <li>
                    Avoid spin-wait for extended periods of time. If possible, do not spin-wait at all.
                </li>
            </ul>
            <li><strong>Network Usage Rules</strong></li>
            <ul>
                <li>
                    Do not use your EdgeNet slice (account) to gain access to any hosting site
                    resources that you did not already have.</li>
                <li>
                    Do not use one or more EdgeNet nodes to flood a site with so much traffic as
                    to interfere with its normal operation. Use congestion-controlled flows for large transfers.
                </li>
                <li>
                    Do not do systematic or random port or address block scans. Do not spoof or sniff traffic.
                </li>
            </ul>
        </ul>
        <h3>Consequences</h3>
        <p>Violation of this Acceptable Use Policy may result in any of the following:</p>
        <ul>
            <li>
                Disabling the slice (account)
            </li>
            <li>
                removing the Site from EdgeNet
            </li>
            <li>
                Informing the organization’s administration.
            </li>
        </ul>
        <p>
            To report a suspected violation of this policy, contact: <a href="mailto:support@planet-lab.eu">support@planet-lab.eu</a>.
        </p>
        <p>
            In case of any breach with this Acceptable Use Policy, Sorbonne Université, on behalf of the PlanetLab
            Europe Steering Committee, shall terminate this Membership Agreement at any time and without written notice
            as provided in the <a href="https://planet-lab.eu/Membership/Agreement">EdgeNet Membership Agreement.</a>
        </p>
    </Box>