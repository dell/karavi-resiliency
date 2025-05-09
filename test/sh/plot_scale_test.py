# Copyright (c) 2025 Dell Inc., or its subsidiaries. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Pre-Req: Install pip, pandas and matplotlib if not already installed
  # yum install -y python3-pip
  # pip3 install pandas matplotlib

import pandas as pd
import matplotlib.pyplot as plt
df = pd.read_csv('recovery_times.csv')
plt.plot(df['num_instances'], df['recovery_time_sec'], marker='o')
plt.title('Number of Instances vs. Time Taken for Recovery')
plt.xlabel('Number of Instances')
plt.ylabel('Recovery Time (Seconds)')
plt.grid(True)
plt.savefig('recovery_graph.png')
